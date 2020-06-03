package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine/health"
	"github.com/kudobuilder/kudo/pkg/engine/renderer"
	parser "github.com/kudobuilder/kudo/pkg/kudoctl/cmd/params"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

// KudoOperatorTask installs an instance of a KUDO operator in a cluster
type KudoOperatorTask struct {
	Name            string
	OperatorName    string
	InstanceName    string
	AppVersion      string
	OperatorVersion string
	ParameterFile   string
}

// Run method for the KudoOperatorTask which will install a child operator
func (kt KudoOperatorTask) Run(ctx Context) (bool, error) {

	// 0. - A few prerequisites -
	// Note: ctx.Meta has Meta.OperatorName and Meta.OperatorVersion fields but these are of the **parent instance**
	// However, since we don't support multiple namespaces yet, we can use the Meta.InstanceNamespace for the namespace
	namespace := ctx.Meta.InstanceNamespace
	operatorName := kt.OperatorName
	operatorVersion := kt.OperatorVersion
	operatorVersionName := v1beta1.OperatorVersionName(operatorName, operatorVersion)
	instanceName := dependencyInstanceName(ctx.Meta.InstanceName, kt.InstanceName, operatorName)

	// 1. - Expand parameter file if exists -
	params, err := instanceParameters(kt.ParameterFile, ctx.Templates, ctx.Meta, ctx.Parameters)
	if err != nil {
		return false, fatalExecutionError(err, taskRenderingError, ctx.Meta)
	}

	// 2. - Build the instance object -
	instance, err := instanceResource(instanceName, operatorName, operatorVersionName, namespace, params, ctx.Meta.ResourcesOwner, ctx.Scheme)
	if err != nil {
		return false, fatalExecutionError(err, taskRenderingError, ctx.Meta)
	}

	// 3. - Apply the Instance object -
	err = applyInstance(instance, namespace, ctx.Client)
	if err != nil {
		return false, err
	}

	// 4. - Check the Instance health -
	if err := health.IsHealthy(instance); err != nil {
		return false, nil
	}

	return true, nil
}

// dependencyInstanceName returns a name for the child instance in an operator with dependencies looking like
// <parent-instance.<child-instance> if a child instance name is provided e.g. `kafka-instance.custom-name` or
// <parent-instance.<child-operator> if not e.g. `kafka-instance.zookeeper`. This way we always have a valid child
// instance name and user can install the same operator multiple times in the same namespace, because the instance
// names will be unique thanks to the top-level instance name prefix.
func dependencyInstanceName(parentInstanceName, instanceName, operatorName string) string {
	if instanceName != "" {
		return fmt.Sprintf("%s.%s", parentInstanceName, instanceName)
	}
	return fmt.Sprintf("%s.%s", parentInstanceName, operatorName)
}

// instanceParameters method takes templated parameter file and a map of parameters and then renders passed template using kudo engine.
func instanceParameters(parameterFile string, templates map[string]string, meta renderer.Metadata, parameters map[string]interface{}) (map[string]string, error) {
	if parameterFile != "" {
		pft, ok := templates[parameterFile]
		if !ok {
			return nil, fmt.Errorf("error finding parameter file %s in templates", parameterFile)
		}

		rendered, err := renderParametersFile(parameterFile, pft, meta, parameters)
		if err != nil {
			return nil, fmt.Errorf("error expanding parameter file %s: %w", parameterFile, err)
		}

		parameters := map[string]string{}
		err = parser.GetParametersFromFile(parameterFile, []byte(rendered), parameters)
		if err != nil {
			return nil, err
		}

		return parameters, nil
	}

	return nil, nil
}

func renderParametersFile(pf string, pft string, meta renderer.Metadata, parameters map[string]interface{}) (string, error) {
	vars := renderer.
		NewVariableMap().
		WithInstance(meta.OperatorName, meta.InstanceName, meta.InstanceNamespace, meta.AppVersion, meta.OperatorVersion).
		WithParameters(parameters)

	engine := renderer.New()

	return engine.Render(pf, pft, vars)
}

func instanceResource(instanceName, operatorName, operatorVersionName, namespace string, parameters map[string]string, owner metav1.Object, scheme *runtime.Scheme) (*v1beta1.Instance, error) {
	instance := &v1beta1.Instance{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Instance",
			APIVersion: packages.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceName,
			Namespace: namespace,
			Labels:    map[string]string{kudo.OperatorLabel: operatorName},
		},
		Spec: v1beta1.InstanceSpec{
			OperatorVersion: corev1.ObjectReference{
				Name: operatorVersionName,
			},
			Parameters: parameters,
		},
		Status: v1beta1.InstanceStatus{},
	}
	if err := controllerutil.SetControllerReference(owner, instance, scheme); err != nil {
		return nil, fmt.Errorf("failed to set resource ownership for the new instance: %v", err)
	}

	return instance, nil
}

// applyInstance creates the passed instance if it doesn't exist or patches the existing one. Patch will override
// current spec.parameters and Spec.operatorVersion the same way, kudoctl does it. If the was no error, then the passed
// instance object is updated with the content returned by the server
func applyInstance(new *v1beta1.Instance, ns string, c client.Client) error {
	old := &v1beta1.Instance{}
	err := c.Get(context.TODO(), types.NamespacedName{Name: new.Name, Namespace: ns}, old)

	switch {
	// 1. if instance doesn't exist, create it
	case apierrors.IsNotFound(err):
		log.Printf("Instance %s/%s doesn't exist. Creating it", new.Namespace, new.Name)
		return createInstance(new, c)
	// 2. if the instance exists (there was no error), try to patch it
	case err == nil:
		log.Printf("Instance %s/%s already exist. Patching it", new.Namespace, new.Name)
		return patchInstance(new, c)
	// 3. any other error is treated as transient
	default:
		return fmt.Errorf("failed to check if instance %s/%s already exists: %v", new.Namespace, new.Name, err)
	}
}

func createInstance(i *v1beta1.Instance, c client.Client) error {
	gvk := i.GroupVersionKind()
	err := c.Create(context.TODO(), i)

	// reset the GVK since it is removed by the c.Create call
	// https://github.com/kubernetes/kubernetes/issues/80609
	i.SetGroupVersionKind(gvk)

	return err
}

func patchInstance(i *v1beta1.Instance, c client.Client) error {
	patch, err := json.Marshal(struct {
		Spec *v1beta1.InstanceSpec `json:"spec"`
	}{
		Spec: &i.Spec,
	})

	if err != nil {
		return fmt.Errorf("failed to serialize instance %s/%s patch: %v", i.Namespace, i.Name, err)
	}

	return c.Patch(context.TODO(), i, client.RawPatch(types.MergePatchType, patch))
}
