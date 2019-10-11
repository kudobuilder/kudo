package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"k8s.io/apimachinery/pkg/types"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	kudoengine "github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/util/health"
	errwrap "github.com/pkg/errors"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apijson "k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type activePlan struct {
	name string
	*v1alpha1.PlanStatus
	spec      *v1alpha1.Plan
	tasks     map[string]v1alpha1.TaskSpec
	templates map[string]string
	params    map[string]string
}

type planResources struct {
	PhaseResources map[string]phaseResources
}

type phaseResources struct {
	StepResources map[string][]runtime.Object
}

type engineMetadata struct {
	instanceName        string
	instanceNamespace   string
	operatorName        string
	operatorVersionName string
	operatorVersion     string

	// the object that will own all the resources created by this execution
	resourcesOwner metav1.Object
}

// executePlan takes a currently active plan and ExecutionMetadata from the underlying operator and executes next "step" in that execution
// the next step could consist of actually executing multiple steps of the plan or just one depending on the execution strategy of the phase (serial/parallel)
// result of running this function is new state of the execution that is returned to the caller (it can either be completed, or still in progress or errored)
// in case of error, error is returned along with the state as well (so that it's possible to report which step caused the error)
// in case of error, method returns ErrorStatus which has property to indicate unrecoverable error meaning if there is no point in retrying that execution
func executePlan(plan *activePlan, metadata *engineMetadata, c client.Client, enhancer kubernetesObjectEnhancer) (*v1alpha1.PlanStatus, error) {
	if plan.Status.IsTerminal() {
		log.Printf("PlanExecution: Plan %s for instance %s is terminal, nothing to do", plan.name, metadata.instanceName)
		return plan.PlanStatus, nil
	}

	// we don't want to modify the original state -> need to do deepcopy
	newState := plan.PlanStatus.DeepCopy()

	// render kubernetes resources needed to execute this plan
	planResources, err := prepareKubeResources(plan, newState, metadata, enhancer)
	if err != nil {
		return newState, err
	}

	// do a next step in the current plan execution
	allPhasesCompleted := true
	for _, ph := range plan.spec.Phases {
		currentPhaseState, _ := getPhaseFromStatus(ph.Name, newState)
		if isFinished(currentPhaseState.Status) {
			// nothing to do
			log.Printf("PlanExecution: Phase %s on plan %s and instance %s is in state %s, nothing to do", ph.Name, plan.name, metadata.instanceName, currentPhaseState.Status)
			continue
		} else if isInProgress(currentPhaseState.Status) {
			newState.Status = v1alpha1.ExecutionInProgress
			currentPhaseState.Status = v1alpha1.ExecutionInProgress
			log.Printf("PlanExecution: Executing phase %s on plan %s and instance %s - it's in progress", ph.Name, plan.name, metadata.instanceName)

			// we're currently executing this phase
			allStepsHealthy := true
			for _, st := range ph.Steps {
				currentStepState, _ := getStepFromStatus(st.Name, currentPhaseState)
				resources := planResources.PhaseResources[ph.Name].StepResources[st.Name]

				log.Printf("PlanExecution: Executing step %s on plan %s and instance %s - it's in %s state", st.Name, plan.name, metadata.instanceName, currentStepState.Status)
				err := executeStep(st, currentStepState, resources, c)
				if err != nil {
					currentPhaseState.Status = v1alpha1.ErrorStatus
					currentStepState.Status = v1alpha1.ErrorStatus
					return newState, err
				}

				if !isFinished(currentStepState.Status) {
					allStepsHealthy = false
					if ph.Strategy == v1alpha1.Serial {
						// we cannot proceed to the next step
						break
					}
				}
			}

			if allStepsHealthy {
				log.Printf("PlanExecution: All steps on phase %s plan %s and instance %s are healthy", ph.Name, plan.name, metadata.instanceName)
				currentPhaseState.Status = v1alpha1.ExecutionComplete
			}
		}

		if !isFinished(currentPhaseState.Status) {
			// we cannot proceed to the next phase
			allPhasesCompleted = false
			break
		}
	}

	if allPhasesCompleted {
		log.Printf("PlanExecution: All phases on plan %s and instance %s are healthy", plan.name, metadata.instanceName)
		newState.Status = v1alpha1.ExecutionComplete
	}

	return newState, nil
}

func executeStep(step v1alpha1.Step, state *v1alpha1.StepStatus, resources []runtime.Object, c client.Client) error {
	if isInProgress(state.Status) {
		state.Status = v1alpha1.ExecutionInProgress

		// check if step is already healthy
		allHealthy := true
		for _, r := range resources {
			if step.Delete {
				// delete
				log.Printf("PlanExecution: Step %s will delete object %v", step.Name, r)
				err := c.Delete(context.TODO(), r, client.PropagationPolicy(metav1.DeletePropagationForeground))
				if !apierrors.IsNotFound(err) && err != nil {
					return err
				}
			} else {
				// create or update
				log.Printf("Going to create/update %v", r)
				existingResource := r.DeepCopyObject()
				key, _ := client.ObjectKeyFromObject(r)
				err := c.Get(context.TODO(), key, existingResource)
				if apierrors.IsNotFound(err) {
					// create
					err = c.Create(context.TODO(), r)
					if err != nil {
						log.Printf("PlanExecution: error when creating resource in step %v: %v", step.Name, err)
						return err
					}
				} else if err != nil {
					// other than not found error - raise it
					return err
				} else {
					// update
					err := patchExistingObject(r, existingResource, c)
					if err != nil {
						return err
					}
				}

				err = health.IsHealthy(c, existingResource)
				if err != nil {
					allHealthy = false
					log.Printf("PlanExecution: Obj is NOT healthy: %s", prettyPrint(key))
				}
			}
		}

		if allHealthy {
			state.Status = v1alpha1.ExecutionComplete
		}
	}
	return nil
}

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "  ")
	return string(s)
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
	_, isUnstructured := newResource.(runtime.Unstructured)
	_, isCRD := newResource.(*apiextv1beta1.CustomResourceDefinition)

	if isUnstructured || isCRD || isKudoType(newResource) {
		// strategic merge patch is not supported for these types, falling back to mergepatch
		err := c.Patch(context.TODO(), newResource, client.ConstantPatch(types.MergePatchType, newResourceJSON))
		if err != nil {
			log.Printf("PlanExecution: Error when applying merge patch to object %v: %v", key, err)
			return err
		}
	} else {
		err := c.Patch(context.TODO(), existingResource, client.ConstantPatch(types.StrategicMergePatchType, newResourceJSON))
		log.Printf("PlanExecution: Error when applying StrategicMergePatch to object %v: %w", key, err)
		return err
	}
	return nil
}

func isKudoType(object runtime.Object) bool {
	_, isOperator := object.(*v1alpha1.OperatorVersion)
	_, isOperatorVersion := object.(*v1alpha1.Operator)
	_, isInstance := object.(*v1alpha1.Instance)
	return isOperator || isOperatorVersion || isInstance
}

// prepareKubeResources takes all resources in all tasks for a plan and renders them with the right parameters
// it also takes care of applying KUDO specific conventions to the resources like commond labels
// newState gets modified with possible state changes as a result of this method
func prepareKubeResources(plan *activePlan, newState *v1alpha1.PlanStatus, meta *engineMetadata, renderer kubernetesObjectEnhancer) (*planResources, error) {
	configs := make(map[string]interface{})
	configs["OperatorName"] = meta.operatorName
	configs["Name"] = meta.instanceName
	configs["Namespace"] = meta.instanceNamespace
	configs["Params"] = plan.params

	result := &planResources{
		PhaseResources: make(map[string]phaseResources),
	}

	for _, phase := range plan.spec.Phases {
		phaseState, _ := getPhaseFromStatus(phase.Name, newState)
		perStepResources := make(map[string][]runtime.Object)
		result.PhaseResources[phase.Name] = phaseResources{
			StepResources: perStepResources,
		}
		for j, step := range phase.Steps {
			configs["planName"] = plan.name
			configs["phaseName"] = phase.Name
			configs["stepName"] = step.Name
			configs["StepNumber"] = strconv.FormatInt(int64(j), 10)
			var resources []runtime.Object
			stepState, _ := getStepFromStatus(step.Name, phaseState)

			engine := kudoengine.New()
			for _, t := range step.Tasks {
				if taskSpec, ok := plan.tasks[t]; ok {
					resourcesAsString := make(map[string]string)

					for _, res := range taskSpec.Resources {
						if resource, ok := plan.templates[res]; ok {
							templatedYaml, err := engine.Render(resource, configs)
							if err != nil {
								newState.Status = v1alpha1.ExecutionFatalError
								phaseState.Status = v1alpha1.ExecutionFatalError
								stepState.Status = v1alpha1.ExecutionFatalError

								err := errwrap.Wrap(err, "error expanding template")
								log.Print(err)
								return nil, &executionError{err, true, nil}
							}
							resourcesAsString[res] = templatedYaml
						} else {
							newState.Status = v1alpha1.ExecutionFatalError
							phaseState.Status = v1alpha1.ExecutionFatalError
							stepState.Status = v1alpha1.ExecutionFatalError

							err := fmt.Errorf("PlanExecution: Error finding resource named %v for operator version %v", res, meta.operatorVersionName)
							log.Print(err)
							return nil, &executionError{err, true, nil}
						}
					}

					resourcesWithConventions, err := renderer.applyConventionsToTemplates(resourcesAsString, ExecutionMetadata{
						engineMetadata: *meta,
						planName:       plan.name,
						phaseName:      phase.Name,
						stepName:       step.Name,
					})

					if err != nil {
						newState.Status = v1alpha1.ErrorStatus
						phaseState.Status = v1alpha1.ErrorStatus
						stepState.Status = v1alpha1.ErrorStatus

						log.Printf("Error creating Kubernetes objects from step %v in phase %v of plan %v and instance %s/%s: %v", step.Name, phase.Name, plan.name, meta.instanceNamespace, meta.instanceName, err)
						return nil, &executionError{err, false, nil}
					}
					resources = append(resources, resourcesWithConventions...)
				} else {
					newState.Status = v1alpha1.ErrorStatus
					phaseState.Status = v1alpha1.ErrorStatus
					stepState.Status = v1alpha1.ErrorStatus

					err := fmt.Errorf("Error finding task named %s for operator version %s", taskSpec, meta.operatorVersionName)
					log.Print(err)
					return nil, &executionError{err, false, nil}
				}
			}

			perStepResources[step.Name] = resources
		}
	}

	return result, nil
}

func getStepFromStatus(stepName string, status *v1alpha1.PhaseStatus) (*v1alpha1.StepStatus, error) {
	for i, p := range status.Steps {
		if p.Name == stepName {
			return &status.Steps[i], nil
		}
	}
	return nil, fmt.Errorf("PlanExecution: Cannot find step %s in plan", stepName)
}

func getPhaseFromStatus(phaseName string, status *v1alpha1.PlanStatus) (*v1alpha1.PhaseStatus, error) {
	for i, p := range status.Phases {
		if p.Name == phaseName {
			return &status.Phases[i], nil
		}
	}
	return nil, fmt.Errorf("PlanExecution: Cannot find phase %s in plan", phaseName)
}

func isFinished(state v1alpha1.ExecutionStatus) bool {
	return state == v1alpha1.ExecutionComplete
}

func isInProgress(state v1alpha1.ExecutionStatus) bool {
	return state == v1alpha1.ExecutionInProgress || state == v1alpha1.ExecutionPending || state == v1alpha1.ErrorStatus
}
