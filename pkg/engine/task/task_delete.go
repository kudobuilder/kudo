package task

import (
	"fmt"

	engine2 "github.com/kudobuilder/kudo/pkg/engine"

	"github.com/kudobuilder/kudo/pkg/controller/instance"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplyTask is a task that attempts to create a set of Kubernetes Resources using a given client
// +k8s:deepcopy-gen=true
type DeleteTask struct {
	Resources []string `json:"deleteResources"`
}

type TaskContext struct {
	Client     client.Client
	Enhancer   instance.KubernetesObjectEnhancer
	Meta       instance.ExecutionMetadata
	Templates  map[string]string // Raw templates
	Parameters map[string]string // I and OV parameters merged
}

// DeleteTask Run method. Given the task context, it renders the templates using context parameters
// creates runtime objects and kustomizes them, and finally removes them using the controller client.
func (c *DeleteTask) Run(ctx TaskContext) (bool, error) {
	// 1. Render task templates
	rendered, err := render(c.Resources, ctx.Templates, ctx.Parameters, ctx.Meta)
	if err != nil {
		return false, err
	}

	// 2. Kustomize them with metadata
	kustomized, err := kustomize(rendered, ctx.Meta, ctx.Enhancer)
	if err != nil {
		return false, err
	}

	// 3. Delete them using the client
	err = delete(kustomized, ctx.Client)
	if err != nil {
		return false, err
	}

	// 4. Check health: always true for Delete task
	return true, nil
}

func render(resourceNames []string, templates map[string]string, params map[string]string, meta instance.ExecutionMetadata) (map[string]string, error) {
	configs := make(map[string]interface{})
	configs["OperatorName"] = meta.OperatorName
	configs["Name"] = meta.InstanceName
	configs["Namespace"] = meta.InstanceNamespace
	configs["Params"] = params

	// TODO (ad) have one metadata object combining instance.Metadata and instance.ExecutionMetaData
	//configs["PlanName"] =
	//configs["PhaseName"] =
	//configs["StepName"] =
	//meta["OwnerRef"] <- add this to Metadata too

	resources := map[string]string{}
	engine := engine2.New()

	for _, rn := range resourceNames {
		resource, ok := templates[rn]

		if !ok {
			return nil, fmt.Errorf("Error finding resource named %v for operator version %v", rn, meta.OperatorVersionName)
		}

		rendered, err := engine.Render(resource, configs)
		if err != nil {
			return nil, errors.Wrap(err, "error expanding template")
		}

		resources[rn] = rendered
	}
	return resources, nil
}

func kustomize(rendered map[string]string, meta instance.ExecutionMetadata, enhancer instance.KubernetesObjectEnhancer) ([]runtime.Object, error) {
	enhanced, err := enhancer.ApplyConventionsToTemplates(rendered, instance.Metadata{
		InstanceName:    meta.InstanceName,
		Namespace:       meta.InstanceNamespace,
		OperatorName:    meta.OperatorName,
		OperatorVersion: meta.OperatorVersion,
		//PlanExecution:   meta.planExecutionID,
		//PlanName:        plan.Name,
		//PhaseName:       phase.Name,
		//StepName:        step.Name,
	}, meta.ResourcesOwner)

	return enhanced, err
}

func delete(objects []runtime.Object, c client.Client) error {
	for _, r := range objects {
		err := c.Delete(context.TODO(), r, client.PropagationPolicy(metav1.DeletePropagationForeground))
		if !apierrors.IsNotFound(err) && err != nil {
			return err
		}
	}

	return nil
}
