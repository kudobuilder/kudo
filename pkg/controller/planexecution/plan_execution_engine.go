package planexecution

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	kudoengine "github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/util/health"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type planState struct {
	Name        string
	State       v1alpha1.PhaseState
	PhasesState map[string]*phaseState
}

type phaseState struct {
	Name       string
	State      v1alpha1.PhaseState
	StepsState map[string]*stepState
}

type stepState struct {
	Name  string
	State v1alpha1.PhaseState
}

type commandType int

const (
	// delete commandType = 0
	update commandType = 1
	create commandType = 2
)

type kubernetesCommand struct {
	Type   commandType
	Obj    runtime.Object
	OldObj runtime.Object
}

type activePlan struct {
	Name string
	Plan *v1alpha1.Plan
}

type planResources struct {
	PhaseResources map[string]phaseResources
}

type phaseResources struct {
	StepResources map[string][]runtime.Object
}

// executePlan ...
// TODO: remove planExecutionId when PE CRD is removed
func executePlan(plan *activePlan, planExecutionID string, currentState *planState, instance *v1alpha1.Instance, params map[string]string, operatorVersion *v1alpha1.OperatorVersion, c client.Client, scheme *runtime.Scheme) (*planState, []*kubernetesCommand, error) {
	if currentState.State == v1alpha1.PhaseStateComplete {
		// nothing to do, plan is already finished
		return currentState, []*kubernetesCommand{}, nil
	}

	// render kubernetes resources needed to execute this plan
	planResources, err := prepareKubeResources(plan, planExecutionID, instance, params, operatorVersion, scheme)
	if err != nil {
		return nil, nil, err
	}

	newState := currentState // TODO deep copy
	outputCommands := make([]*kubernetesCommand, 0)
	// do a next step in the current plan execution
	for _, ph := range plan.Plan.Phases {
		currentPhaseState := newState.PhasesState[ph.Name]
		if currentPhaseState.State == v1alpha1.PhaseStateComplete || currentPhaseState.State == v1alpha1.PhaseStateError {
			// nothing to do
			continue
		}
		if currentPhaseState.State == v1alpha1.PhaseStateInProgress || currentPhaseState.State == v1alpha1.PhaseStatePending {
			currentPhaseState.State = v1alpha1.PhaseStateInProgress

			// we're currently executing this phase
			for _, s := range ph.Steps {
				currentStepState := currentPhaseState.StepsState[s.Name]
				resources := planResources.PhaseResources[ph.Name].StepResources[s.Name]
				if currentStepState.State == v1alpha1.PhaseStateInProgress {
					// check if step is already healthy
					allHealthy := true
					for _, r := range resources {
						key, _ := client.ObjectKeyFromObject(r)
						err := c.Get(context.TODO(), key, r)
						if err != nil {
							return nil, nil, err
						}

						err = health.IsHealthy(c, r)
						if err != nil {
							allHealthy = false
							log.Printf("PlanExecutionController: Obj is NOT healthy: %s", prettyPrint(key))
						}
					}

					if allHealthy {
						currentStepState.State = v1alpha1.PhaseStateComplete
					}
				} else if currentStepState.State == v1alpha1.PhaseStatePending {
					// we need to create or update the resource
					for _, r := range resources {
						currentStepState.State = v1alpha1.PhaseStateInProgress

						oldObj := r.DeepCopyObject()
						key, _ := client.ObjectKeyFromObject(r)
						err := c.Get(context.TODO(), key, oldObj)
						if apierrors.IsNotFound(err) {
							outputCommands = append(outputCommands, &kubernetesCommand{
								Type: create,
								Obj:  r,
							})
						} else if err != nil { // other that not found error
							return nil, nil, err
						} else {
							outputCommands = append(outputCommands, &kubernetesCommand{
								Type:   update,
								Obj:    r,
								OldObj: oldObj,
							})
						}
					}
				}

				if currentStepState.State != v1alpha1.PhaseStateComplete {
					break
				}
			}

			if currentPhaseState.State != v1alpha1.PhaseStateComplete {
				break
			}
		}
	}

	return newState, outputCommands, nil
}

// prepareKubeResources takes all resources in all tasks for a plan and renders them with the right parameters
// it also takes care of applying KUDO specific conventions to the resources like commond labels
func prepareKubeResources(activePlan *activePlan, planExecutionID string, instance *v1alpha1.Instance, params map[string]string, operatorVersion *v1alpha1.OperatorVersion, scheme *runtime.Scheme) (*planResources, error) {
	configs := make(map[string]interface{})
	configs["OperatorName"] = operatorVersion.Spec.Operator.Name
	configs["Name"] = instance.Name
	configs["Namespace"] = instance.Namespace
	configs["Params"] = params

	result := &planResources{
		PhaseResources: make(map[string]phaseResources),
	}

	for _, phase := range activePlan.Plan.Phases {
		perStepResources := make(map[string][]runtime.Object)
		result.PhaseResources[phase.Name] = phaseResources{
			StepResources: perStepResources,
		}
		for j, step := range phase.Steps {
			configs["PlanName"] = activePlan.Name
			configs["PhaseName"] = phase.Name
			configs["StepName"] = step.Name
			configs["StepNumber"] = strconv.FormatInt(int64(j), 10)
			var resources []runtime.Object

			engine := kudoengine.New()
			for _, t := range step.Tasks {
				if taskSpec, ok := operatorVersion.Spec.Tasks[t]; ok {
					resourcesAsString := make(map[string]string)

					for _, res := range taskSpec.Resources {
						if resource, ok := operatorVersion.Spec.Templates[res]; ok {
							templatedYaml, err := engine.Render(resource, configs)
							if err != nil {
								err := errors.Wrapf(err, "error expanding template")
								log.Print(err)
								return nil, fatalError{err: err}
							}
							resourcesAsString[res] = templatedYaml
						} else {
							err := fmt.Errorf("PlanExecutionController: Error finding resource named %v for operator version %v", res, operatorVersion.Name)
							log.Print(err)
							return nil, fatalError{err: err}
						}
					}

					resourcesWithConventions, err := applyConventionsToTemplates(resourcesAsString, metadata{
						InstanceName:    instance.Name,
						Namespace:       instance.Namespace,
						OperatorName:    operatorVersion.Spec.Operator.Name,
						OperatorVersion: operatorVersion.Spec.Version,
						PlanExecution:   planExecutionID,
						PlanName:        activePlan.Name,
						PhaseName:       phase.Name,
						StepName:        step.Name,
					}, instance, scheme)

					if err != nil {
						log.Printf("Error creating Kubernetes objects from step %v in phase %v of plan %v: %v", step.Name, phase.Name, planExecutionID, err)
						return nil, err
					}
					resources = append(resources, resourcesWithConventions...)
				} else {
					err := fmt.Errorf("Error finding task named %s for operator version %s", taskSpec, operatorVersion.Name)
					log.Print(err)
					return nil, fatalError{err: err}
				}
			}

			perStepResources[step.Name] = resources
		}
	}

	return result, nil
}
