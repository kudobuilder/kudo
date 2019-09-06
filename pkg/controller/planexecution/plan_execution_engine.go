package planexecution

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"k8s.io/apimachinery/pkg/types"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	kudoengine "github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/util/health"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	apijson "k8s.io/apimachinery/pkg/util/json"
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
func executePlan(plan *activePlan, planExecutionID string, currentState *planState, instance *v1alpha1.Instance, params map[string]string, operatorVersion *v1alpha1.OperatorVersion, c client.Client, scheme *runtime.Scheme) (*planState, error) {
	if currentState.State == v1alpha1.PhaseStateComplete {
		// nothing to do, plan is already finished
		return currentState, nil
	}

	// render kubernetes resources needed to execute this plan
	planResources, err := prepareKubeResources(plan, planExecutionID, instance, params, operatorVersion, scheme)
	if err != nil {
		return nil, err
	}

	newState := currentState // TODO deep copy
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

				err := executeStep(s, currentPhaseState, resources, c)
				if err != nil {
					return nil, err
				}

				if currentStepState.State != v1alpha1.PhaseStateComplete {
					// we cannot proceed to the next step
					break
				}
			}

			if currentPhaseState.State != v1alpha1.PhaseStateComplete {
				// we cannot proceed to the next phase
				break
			}
		}
	}

	return newState, nil
}

func executeStep(step v1alpha1.Step, state *phaseState, resources []runtime.Object, c client.Client) error {
	if state.State == v1alpha1.PhaseStateInProgress || state.State == v1alpha1.PhaseStatePending {
		// check if step is already healthy
		allHealthy := true
		for _, r := range resources {
			existingResource := r.DeepCopyObject()
			key, _ := client.ObjectKeyFromObject(r)
			err := c.Get(context.TODO(), key, existingResource)
			if apierrors.IsNotFound(err) {
				err = c.Create(context.TODO(), r)
				if err != nil {
					log.Printf("PlanExecution: error when creating resource in step %v: %v", step.Name, err)
					return err
				}
			} else if err != nil { // other than not found error - raise it
				return err
			} else {
				// try to update the resource
				err := patchExistingObject(r, existingResource, c)
				if err != nil {
					return err
				}
			}

			err = health.IsHealthy(c, r)
			if err != nil {
				allHealthy = false
				log.Printf("PlanExecution: Obj is NOT healthy: %s", prettyPrint(key))
			}
		}

		if allHealthy {
			state.State = v1alpha1.PhaseStateComplete
		}
	}
	return nil
}

// patchExistingObject calls update method on kubernetes client to make sure the current resource reflects what is on server
//
// an obvious optimization here would be to not patch when objects are the same, however that is not easy
// kubernetes native objects might be a problem because we cannot just compare the spec as the spec might have extra fields
// and those extra fields are set by some kubernetes component
// because of that for now we just try to apply the patch every time
func patchExistingObject(newResource runtime.Object, existingResource runtime.Object, c client.Client) error {
	newResourceJSON, _ := apijson.Marshal(newResource)
	key, _ := client.ObjectKeyFromObject(newResource)
	err := c.Patch(context.TODO(), existingResource, client.ConstantPatch(types.StrategicMergePatchType, newResourceJSON))
	if err != nil {
		// Right now applying a Strategic Merge Patch to custom resources does not work. There is
		// certain metadata needed, which when missing, leads to an invalid Content-Type Header and
		// causes the request to fail.
		// ( see https://github.com/kubernetes-sigs/kustomize/issues/742#issuecomment-458650435 )
		//
		// We temporarily solve this by checking for the specific error when a SMP is applied to
		// custom resources and handle it by defaulting to a Merge Patch.
		//
		// The error message for which we check is:
		// 		the body of the request was in an unknown format - accepted media types include:
		//			application/json-patch+json, application/merge-patch+json
		//
		// 		Reason: "UnsupportedMediaType" Code: 415
		if apierrors.IsUnsupportedMediaType(err) {
			err = c.Patch(context.TODO(), newResource, client.ConstantPatch(types.MergePatchType, newResourceJSON))
			if err != nil {
				log.Printf("PlanExecution: Error when applying merge patch to object %v: %v", key, err)
				return err
			}
		} else {
			log.Printf("PlanExecution: Error when applying StrategicMergePatch to object %v: %v", key, err)
			return err
		}
	}
	return nil
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
							err := fmt.Errorf("PlanExecution: Error finding resource named %v for operator version %v", res, operatorVersion.Name)
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
