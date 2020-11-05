package kudo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	v1core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	// Import Kubernetes authentication providers to support GKE, etc.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/scheme"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/crd"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
	"github.com/kudobuilder/kudo/pkg/util/convert"
	label "github.com/kudobuilder/kudo/pkg/util/kudo"
	"github.com/kudobuilder/kudo/pkg/version"
)

// Client is a KUDO Client providing access to a kudo clientset and kubernetes clientsets
type Client struct {
	kudoClientset versioned.Interface
	KubeClientset kubernetes.Interface
}

// NewClient creates new KUDO Client
func NewClient(kubeConfigPath string, requestTimeout int64, validateInstall bool) (*Client, error) {

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}

	// set default configs
	config.Timeout = time.Duration(requestTimeout) * time.Second

	return NewClientForConfig(config, validateInstall)
}

// NewClient creates new KUDO Client
func NewClientForConfig(config *rest.Config, validateInstall bool) (*Client, error) {
	kubeClient, err := kube.GetKubeClientForConfig(config)
	if err != nil {
		return nil, clog.Errorf("could not get Kubernetes client: %s", err)
	}

	// create the kudo clientset
	kudoClientset, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// create the kubernetes clientset
	client := &Client{
		kudoClientset: kudoClientset,
		KubeClientset: kubeClient.KubeClient,
	}

	if validateInstall {
		if err := client.VerifyServedCRDs(kubeClient); err != nil {
			return nil, err
		}
	}

	return client, nil
}

// NewClientFromK8s creates KUDO client from kubernetes client interface
func NewClientFromK8s(kudo versioned.Interface, kube kubernetes.Interface) *Client {
	result := Client{}
	result.kudoClientset = kudo
	result.KubeClientset = kube
	return &result
}

func (c *Client) VerifyServedCRDs(kubeClient *kube.Client) error {
	result := verifier.NewResult()
	err := crd.NewInitializer().VerifyServedVersion(kubeClient, kudoapi.SchemeGroupVersion.Version, &result)
	if err != nil {
		return fmt.Errorf("failed to run crd verification: %v", err)
	}
	if !result.IsValid() {
		clog.V(0).Printf("KUDO CRDs are not served in the expected version.")
		return fmt.Errorf("CRDs invalid: %v", result.ErrorsAsString())
	}

	return nil
}

// OperatorExistsInCluster checks if a given Operator object is installed on the current k8s cluster
func (c *Client) OperatorExistsInCluster(name, namespace string) bool {
	operator, err := c.kudoClientset.KudoV1beta1().Operators(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		clog.V(2).Printf("operator.kudo.dev %s/%s does not exist\n", namespace, name)
		return false
	}
	clog.V(2).Printf("operator.kudo.dev %s/%s unchanged", operator.Namespace, operator.Name)
	return true
}

// OperatorVersionExistsInCluster checks if a given OperatorVersion object is installed on the current k8s cluster
func (c *Client) OperatorVersionExistsInCluster(name, namespace string) bool {
	operator, err := c.kudoClientset.KudoV1beta1().OperatorVersions(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if err != nil {
		clog.V(2).Printf("operatorversion.kudo.dev %s/%s does not exist\n", namespace, name)
		return false
	}
	clog.V(2).Printf("operatorversion.kudo.dev %s/%s unchanged", operator.Namespace, operator.Name)
	return true
}

// InstanceExistsInCluster checks if any OperatorVersion object matches to the given Operator name
// in the cluster.
// An Instance has two identifiers:
// 		1) Spec.OperatorVersion.Name
// 		spec:
//    		operatorVersion:
//      		name: kafka-2.11-2.4.0
// 		2) LabelSelector
// 		metadata:
//    		creationTimestamp: "2019-02-28T14:39:20Z"
//    		generation: 1
//    		labels:
//      		kudo.dev/operator: kafka
// This function also just returns true if the Instance matches a specific OperatorVersion of an Operator
func (c *Client) InstanceExistsInCluster(operatorName, namespace, version, instanceName string) (bool, error) {
	instances, err := c.kudoClientset.KudoV1beta1().Instances(namespace).List(context.TODO(), v1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", label.OperatorLabel, operatorName)})
	if err != nil {
		return false, err
	}
	if len(instances.Items) < 1 {
		return false, nil
	}

	// TODO: check function that actual checks for the OperatorVersion named e.g. "test-1.0" to exist
	var i int
	for _, v := range instances.Items {
		if v.Spec.OperatorVersion.Name == operatorName+"-"+version && v.ObjectMeta.Name == instanceName {
			i++
		}
	}

	// No instance exist with this operatorName and OV exists
	if i == 0 {
		return false, nil
	}
	return true, nil
}

// Populate the GVK from scheme, since it is cleared by design on typed objects.
// https://github.com/kubernetes/client-go/issues/413
func SetGVKFromScheme(object runtime.Object, scheme *runtime.Scheme) error {
	gvks, unversioned, err := scheme.ObjectKinds(object)
	if err != nil {
		return err
	}
	if len(gvks) == 0 {
		return fmt.Errorf("no ObjectKinds available for %T", object)
	}
	if !unversioned {
		object.GetObjectKind().SetGroupVersionKind(gvks[0])
	}
	return nil
}
func setGVKFromScheme(object runtime.Object) error {
	return SetGVKFromScheme(object, scheme.Scheme)
}

// GetInstance queries kubernetes api for instance of given name in given namespace
// returns error for error conditions. Instance not found is not considered an error and will result in 'nil, nil'
func (c *Client) GetInstance(name, namespace string) (*kudoapi.Instance, error) {
	instance, err := c.kudoClientset.KudoV1beta1().Instances(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return instance, err
	}
	err = setGVKFromScheme(instance)
	return instance, err
}

// GetOperatorVersion queries kubernetes api for operatorversion of given name in given namespace
// returns error for all other errors that not found, not found is treated as result being 'nil, nil'
func (c *Client) GetOperatorVersion(name, namespace string) (*kudoapi.OperatorVersion, error) {
	ov, err := c.kudoClientset.KudoV1beta1().OperatorVersions(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return ov, err
	}
	err = setGVKFromScheme(ov)
	return ov, err
}

// GetOperatorVersion queries kubernetes api for operator of given name in given namespace
// returns error for all other errors that not found, not found is treated as result being 'nil, nil'
func (c *Client) GetOperator(name, namespace string) (*kudoapi.Operator, error) {
	o, err := c.kudoClientset.KudoV1beta1().Operators(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return o, fmt.Errorf("failed to get operator %s/%s: %v", namespace, name, err)
	}
	err = setGVKFromScheme(o)
	return o, err
}

// UpdateInstance updates operatorversion on instance
func (c *Client) UpdateInstance(instanceName, namespace string, operatorVersion *string, parameters map[string]string, triggeredPlan *string, wait bool, waitTime time.Duration) error {
	var oldInstance *kudoapi.Instance
	if wait {
		var err error
		oldInstance, err = c.GetInstance(instanceName, namespace)
		if err != nil {
			return err
		}
	}

	instanceSpec := kudoapi.InstanceSpec{}
	// 1. new OperatorVersion
	if operatorVersion != nil {
		instanceSpec.OperatorVersion = v1core.ObjectReference{
			Name: convert.StringValue(operatorVersion),
		}
	}
	// 2. new/updated parameters
	if parameters != nil {
		instanceSpec.Parameters = parameters
	}
	// 3. new/updated execution plan
	if triggeredPlan != nil {
		instanceSpec.PlanExecution.PlanName = *triggeredPlan
		instanceSpec.PlanExecution.UID = uuid.NewUUID() // we need to generate a new UID for KUDO manager to detect a new plan
	}

	// 4. create new instance object and patch the existing one
	serializedPatch, err := json.Marshal(struct {
		Spec *kudoapi.InstanceSpec `json:"spec"`
	}{
		&instanceSpec,
	})
	if err != nil {
		return err
	}
	_, err = c.kudoClientset.KudoV1beta1().Instances(namespace).Patch(context.TODO(), instanceName, types.MergePatchType, serializedPatch, v1.PatchOptions{})
	if err != nil {
		return err
	}
	if !wait {
		return nil
	}
	return c.WaitForInstance(instanceName, namespace, oldInstance, waitTime)
}

// WaitForInstance waits for instance to be "complete".
// It uses controller-runtime `wait.PollImmediate`, the function passed to it returns done==false if it isn't done.
// For a situation where there is no previous state (like install), the "lastPlanStatus" will be nil until the manager
// sets it, then it's state will be watched (see InInstanceDone for more detail)
// For a situation where there is previous state (like update, upgrade, plan trigger) than it is important AND required
// that the "oldInstance" be provided.  Without it, it is possible for this function to be "racy" and "flaky" meaning the
// "current" status could be the old "done" status or the new status... it's hard to know.  If the oldInstance is provided
// the wait will then initially wait for the "new" plan to activate then return when completed.
// The error is either an error in working with kubernetes or a wait.ErrWaitTimeout
func (c *Client) WaitForInstance(name, namespace string, oldInstance *kudoapi.Instance, timeout time.Duration) error {
	// polling interval 1 sec
	interval := 1 * time.Second
	return wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		instance, err := c.GetInstance(name, namespace)
		if err != nil {
			return false, err
		}

		return c.IsInstanceDone(instance, oldInstance)
	})
}

// WaitForInstanceDeleted waits for instance to be removed from the cluster.
func (c *Client) WaitForInstanceDeleted(name, namespace string, timeout time.Duration) error {
	interval := 1 * time.Second

	return wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		instance, err := c.GetInstance(name, namespace)
		if err != nil {
			return false, err
		}

		if instance == nil {
			clog.V(2).Printf("instance %q was deleted\n", name)
			return true, nil
		}

		clog.V(4).Printf("instance %q is still running\n", name)
		return false, nil
	})
}

// IsInstanceDone provides a check on instance to see if it is "finished" without retries
// oldInstance is nil if there is no previous instance
func (c *Client) IsInstanceDone(instance, oldInstance *kudoapi.Instance) (bool, error) {

	// upgrade wait, needs to make sure the UID switches
	if oldInstance != nil {
		// We want one of the plans UIDs to change to identify that a new plan ran.
		// If they're all the same, then nothing changed.
		same := true
		for planName, planStatus := range oldInstance.Status.PlanStatus {
			same = same && planStatus.UID == instance.Status.PlanStatus[planName].UID
		}
		if same {
			// Nothing changed yet... waiting on the right plan to wait on
			return false, nil
		}
	}
	lastPlanStatus := instance.GetLastExecutedPlanStatus()
	// must have a status to check
	if lastPlanStatus == nil {
		clog.V(2).Printf("plan status for instance %q is not available\n", instance.Name)
		return false, nil
	}
	status := lastPlanStatus.Status
	if status.IsFinished() {
		clog.V(2).Printf("plan status for %q is finished\n", instance.Name)
		return true, nil
	}

	clog.V(4).Printf("\rinstance plan %q is not not finished running: %v, term: %v, finished: %v", lastPlanStatus.Name, status.IsRunning(), status.IsTerminal(), status.IsFinished())
	return false, nil
}

// IsInstanceByNameDone provides a check on instance based on name to see if it is "finished" without retries
// returns true if finished otherwise false
// oldInstance is nil if there is no previous instance
func (c *Client) IsInstanceByNameDone(name string, namespace string, oldInstance *kudoapi.Instance) (bool, error) {
	instance, err := c.GetInstance(name, namespace)
	if err != nil {
		return false, err
	}

	return c.IsInstanceDone(instance, oldInstance)
}

// ListInstancesAsRuntimeObject lists all instances installed in the cluster in a given ns
func (c *Client) ListInstancesAsRuntimeObject(namespace string) ([]runtime.Object, error) {
	instances, err := c.kudoClientset.KudoV1beta1().Instances(namespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	existingItems := []runtime.Object{}
	for i := range instances.Items {
		existingItems = append(existingItems, &instances.Items[i])
	}
	return existingItems, nil
}

func (c *Client) ListOperatorVersions(namespace string) ([]kudoapi.OperatorVersion, error) {
	ovs, err := c.kudoClientset.KudoV1beta1().OperatorVersions(namespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return ovs.Items, nil
}

// ListOperatorVersionsAsRuntimeObject lists all operatorversions installed in the cluster in a given ns
func (c *Client) ListOperatorVersionsAsRuntimeObject(namespace string) ([]runtime.Object, error) {
	ovs, err := c.ListOperatorVersions(namespace)
	if err != nil {
		return nil, err
	}

	asObjs := []runtime.Object{}
	for i := range ovs {
		asObjs = append(asObjs, &ovs[i])
	}
	return asObjs, nil
}

// ListOperatorsAsRuntimeObject lists all operators installed in the cluster in a given ns
func (c *Client) ListOperatorsAsRuntimeObject(namespace string) ([]runtime.Object, error) {
	operators, err := c.kudoClientset.KudoV1beta1().Operators(namespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	existingItems := []runtime.Object{}
	for i := range operators.Items {
		existingItems = append(existingItems, &operators.Items[i])
	}
	return existingItems, nil
}

// OperatorVersionsInstalled lists all the versions of given operator installed in the cluster in given ns
func (c *Client) OperatorVersionsInstalled(operatorName, namespace string) ([]string, error) {
	ov, err := c.kudoClientset.KudoV1beta1().OperatorVersions(namespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	existingVersions := []string{}

	for _, v := range ov.Items {
		if v.Spec.Operator.Name == operatorName {
			existingVersions = append(existingVersions, v.Spec.Version)
		}
	}
	return existingVersions, nil
}

// InstallOperatorObjToCluster expects a valid Operator obj to install
func (c *Client) InstallOperatorObjToCluster(obj *kudoapi.Operator, namespace string) (*kudoapi.Operator, error) {
	createdObj, err := c.kudoClientset.KudoV1beta1().Operators(namespace).Create(context.TODO(), obj, v1.CreateOptions{})
	if err != nil {
		// we do NOT wrap timeouts
		if os.IsTimeout(err) {
			return nil, err
		}
		return nil, fmt.Errorf("installing Operator: %w", err)
	}
	return createdObj, nil
}

// InstallOperatorVersionObjToCluster expects a valid Operator obj to install
func (c *Client) InstallOperatorVersionObjToCluster(obj *kudoapi.OperatorVersion, namespace string) (*kudoapi.OperatorVersion, error) {
	createdObj, err := c.kudoClientset.KudoV1beta1().OperatorVersions(namespace).Create(context.TODO(), obj, v1.CreateOptions{})
	if err != nil {
		// we do NOT wrap timeouts
		if os.IsTimeout(err) {
			return nil, err
		}
		return nil, fmt.Errorf("installing OperatorVersion: %w", err)
	}
	return createdObj, nil
}

// InstallInstanceObjToCluster expects a valid Instance obj to install
func (c *Client) InstallInstanceObjToCluster(obj *kudoapi.Instance, namespace string) (*kudoapi.Instance, error) {
	createdObj, err := c.kudoClientset.KudoV1beta1().Instances(namespace).Create(context.TODO(), obj, v1.CreateOptions{})
	if err != nil {
		// we do NOT wrap timeouts
		if os.IsTimeout(err) {
			return nil, err
		}
		return nil, fmt.Errorf("installing Instance: %w", err)
	}
	clog.V(2).Printf("instance %v created in namespace %v", createdObj.Name, namespace)
	return createdObj, nil
}

// DeleteInstance deletes an instance.
func (c *Client) DeleteInstance(instanceName, namespace string) error {
	propagationPolicy := v1.DeletePropagationBackground
	options := v1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	}

	return c.kudoClientset.KudoV1beta1().Instances(namespace).Delete(context.TODO(), instanceName, options)
}

// ValidateServerForOperator validates that the k8s server version and kudo version are valid for operator
// error message will provide detail of failure, otherwise nil
func (c *Client) ValidateServerForOperator(operator *kudoapi.Operator) error {
	expectedKubver, err := version.New(operator.Spec.KubernetesVersion)
	if err != nil {
		return fmt.Errorf("unable to parse operators kubernetes version: %w", err)
	}
	// TODO : to be added in when we support kudo server providing server version
	// expectedKudoVer, err := semver.NewVersion(operator.Spec.KudoVersion)
	// if err != nil {
	// 	return fmt.Errorf("Unable to parse operators kudo version: %w", err)
	// }
	// semvar compares patch, for which we do not want to... compare maj, min only
	kVer, err := getKubeVersion(c.kudoClientset.Discovery())
	if err != nil {
		return err
	}

	kSemVer, err := version.FromGithubVersion(kVer)
	if err != nil {
		return err
	}
	if expectedKubver.CompareMajorMinor(kSemVer) > 0 {
		return fmt.Errorf("expected kubernetes version of %v is not supported with version: %v", expectedKubver, kSemVer)
	}

	return nil
}

func (c *Client) CreateNamespace(namespace, manifest string) error {

	ns := &v1core.Namespace{}
	if manifest != "" {
		if err := yaml.Unmarshal([]byte(manifest), ns); err != nil {
			return fmt.Errorf("unmarshalling namespace manifest file: %w", err)
		}
	}
	ns.TypeMeta.Kind = "Namespace"
	ns.Name = namespace

	if ns.Annotations == nil {
		ns.Annotations = map[string]string{}
	}
	ns.Annotations["created-by"] = "kudo-cli"

	_, err := c.KubeClientset.CoreV1().Namespaces().Create(context.TODO(), ns, v1.CreateOptions{})
	return err
}

// GetChildInstances returns all instances that were created as dependencies of a parent instance
func (c *Client) GetChildInstances(parent *kudoapi.Instance) ([]kudoapi.Instance, error) {
	instances, err := c.kudoClientset.KudoV1beta1().Instances(parent.Namespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	children := []kudoapi.Instance{}

	for _, instance := range instances.Items {
		for _, or := range instance.GetOwnerReferences() {
			if parent.UID == or.UID {
				children = append(children, instance)
			}
		}
	}

	return children, nil
}

// getKubeVersion returns stringified version of k8s server
func getKubeVersion(client discovery.ServerVersionInterface) (string, error) {
	v, err := client.ServerVersion()
	if err != nil {
		return "", err
	}
	return v.String(), nil
}
