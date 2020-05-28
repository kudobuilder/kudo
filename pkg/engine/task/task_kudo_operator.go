package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
	Package         string
	InstanceName    string
	AppVersion      string
	OperatorVersion string
	ParameterFile   string
}

// Run method for the KudoOperatorTask. Not yet implemented
func (dt KudoOperatorTask) Run(ctx Context) (bool, error) {

	// 0. - A few prerequisites -
	// Note: ctx.Meta has Meta.OperatorName and Meta.OperatorVersion fields but these are of the **parent instance**
	// However, since we don't support multiple namespaces yet, we can use the Meta.InstanceNamespace for the namespace
	namespace := ctx.Meta.InstanceNamespace
	operatorName := dt.Package
	operatorVersion := dt.OperatorVersion
	operatorVersionName := v1beta1.OperatorVersionName(operatorName, operatorVersion)

	// 1. - Expand parameter file if exists -
	params, err := instanceParameters(dt.ParameterFile, ctx.Templates, ctx.Meta, ctx.Parameters)
	if err != nil {
		return false, fatalExecutionError(err, taskRenderingError, ctx.Meta)
	}

	// 2. - Build the instance object -
	// TODO: make it possible to install the same operator N times in the same namespace by making instance names unique/hierarchical
	instance := instanceResource(operatorName, operatorVersionName, namespace, params)

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

// render method takes templated parameter file and a map of parameters and then renders passed template using kudo engine.
func instanceParameters(pf string, templates map[string]string, meta renderer.Metadata, parameters map[string]interface{}) (map[string]string, error) {
	if len(pf) != 0 {
		pft, ok := templates[pf]
		if !ok {
			return nil, fmt.Errorf("error finding parameter file %s", pf)
		}

		rendered, err := renderParametersFile(pf, pft, meta, parameters)
		if err != nil {
			return nil, fmt.Errorf("error expanding parameter file %s: %w", pf, err)
		}

		parameters := map[string]string{}
		errs := []string{}
		parser.GetParametersFromFile(pf, []byte(rendered), errs, parameters)
		if len(errs) > 0 {
			return nil, fmt.Errorf("failed to unmarshal parameter file %s: %s", pf, strings.Join(errs, ", "))
		}

		return parameters, nil
	}

	return nil, nil
}

func renderParametersFile(pf string, pft string, meta renderer.Metadata, parameters map[string]interface{}) (string, error) {
	configs := make(map[string]interface{})
	configs["OperatorName"] = meta.OperatorName
	configs["AppVersion"] = meta.AppVersion
	configs["Name"] = meta.InstanceName
	configs["Namespace"] = meta.InstanceNamespace
	configs["Params"] = parameters

	engine := renderer.New()

	return engine.Render(pf, pft, configs)
}

func instanceResource(operatorName, operatorVersionName, namespace string, parameters map[string]string) *v1beta1.Instance {
	return &v1beta1.Instance{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Instance",
			APIVersion: packages.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      v1beta1.InstanceName(operatorName),
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
