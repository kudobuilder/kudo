package task

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/engine"

	"golang.org/x/net/context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeleteTask will delete a set of given resources from the cluster. See Run method for more details.
type DeleteTask struct {
	Name      string
	Resources []string
}

// DeleteTask Run method. Given the task context, it renders the templates using context parameters
// creates runtime objects and kustomizes them, and finally removes them using the controller client.
func (dt DeleteTask) Run(ctx Context) (bool, error) {
	// 1. Render task templates
	rendered, err := render(dt.Resources, ctx.Templates, ctx.Parameters, ctx.Meta)
	if err != nil {
		return false, fmt.Errorf("%wfailed to render task resources: %v", FatalExecutionError, err)
	}

	// 2. Kustomize them with metadata
	kustomized, err := kustomize(rendered, ctx.Meta, ctx.Enhancer)
	if err != nil {
		return false, fmt.Errorf("%wfailed to kustomize task resources: %v", FatalExecutionError, err)
	}

	// 3. Delete them using the client
	err = delete(kustomized, ctx.Client)
	if err != nil {
		return false, err
	}

	// 4. Check health: always true for Delete task
	return true, nil
}

// TODO(ad) find a better place for this method
// render method takes resource names and Instance parameters and then renders passed templates using kudo engine.
func render(resourceNames []string, templates map[string]string, params map[string]string, meta ExecutionMetadata) (map[string]string, error) {
	configs := make(map[string]interface{})
	configs["OperatorName"] = meta.OperatorName
	configs["Name"] = meta.InstanceName
	configs["Namespace"] = meta.InstanceNamespace
	configs["Params"] = params
	configs["PlanName"] = meta.PlanName
	configs["PhaseName"] = meta.PhaseName
	configs["StepName"] = meta.StepName

	resources := map[string]string{}
	engine := engine.New()

	for _, rn := range resourceNames {
		resource, ok := templates[rn]

		if !ok {
			return nil, fmt.Errorf("error finding resource named %v for operator version %v", rn, meta.OperatorVersionName)
		}

		rendered, err := engine.Render(resource, configs)
		if err != nil {
			return nil, fmt.Errorf("error expanding template: %w", err)
		}

		resources[rn] = rendered
	}
	return resources, nil
}

// TODO (ad) find a better place for this method
// kustomize method takes a slice of rendered templates, applies conventions using KubernetesObjectEnhancer and
// returns a slice of k8s objects.
func kustomize(rendered map[string]string, meta ExecutionMetadata, enhancer KubernetesObjectEnhancer) ([]runtime.Object, error) {
	enhanced, err := enhancer.ApplyConventionsToTemplates(rendered, meta)
	return enhanced, err
}

func delete(ro []runtime.Object, c client.Client) error {
	for _, r := range ro {
		err := c.Delete(context.TODO(), r, client.PropagationPolicy(metav1.DeletePropagationForeground))
		if !apierrors.IsNotFound(err) && err != nil {
			return err
		}
	}

	return nil
}
