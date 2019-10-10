package instance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	engtask "github.com/kudobuilder/kudo/pkg/engine/task"
	"k8s.io/apimachinery/pkg/types"

	kudoengine "github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/util/health"
	errwrap "github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apijson "k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	UnknownTaskNameEventName         = "UnknownTaskName"
	UnknownTaskKindEventName         = "UnknownTaskName"
	TaskExecutionErrorEventName      = "TaskExecutionError"
	FatalTaskExecutionErrorEventName = "FatalTaskExecutionError"
)

type activePlan struct {
	name string
	*v1alpha1.PlanStatus
	spec      *v1alpha1.Plan
	tasks     []v1alpha1.Task
	templates map[string]string
	params    map[string]string
}

func (ap *activePlan) taskByName(name string) (*v1alpha1.Task, bool) {
	for _, t := range ap.tasks {
		if t.Name == name {
			return &t, true
		}
	}
	return nil, false
}

type planResources struct {
	PhaseResources map[string]phaseResources
}

type phaseResources struct {
	StepResources map[string][]runtime.Object
}

type EngineMetadata struct {
	InstanceName        string
	InstanceNamespace   string
	OperatorName        string
	OperatorVersionName string
	OperatorVersion     string

	// the object that will own all the resources created by this execution
	ResourcesOwner metav1.Object
}

// === New plan execution entry point ===
func executePlan2(pl *activePlan, em *EngineMetadata, c client.Client, enh KubernetesObjectEnhancer) (*v1alpha1.PlanStatus, error) {
	if pl.Status.IsTerminal() {
		log.Printf("PlanExecution: Plan %s for instance %s is terminal, nothing to do", pl.name, em.InstanceName)
		return pl.PlanStatus, nil
	}

	planStatus := pl.PlanStatus.DeepCopy()
	planStatus.Status = v1alpha1.ExecutionInProgress

	phasesLeft := len(pl.spec.Phases)
	// --- 1. Iterate over plan phases ---
	for _, ph := range pl.spec.Phases {
		phaseStatus := getOrCreatePhaseStatus(ph.Name, planStatus)

		// Check current phase status: skip if finished, proceed if in progress, break out if a fatal error has occurred
		if isFinished(phaseStatus.Status) {
			phasesLeft = phasesLeft - 1
			continue
		} else if isInProgress(phaseStatus.Status) {
			phaseStatus.Status = v1alpha1.ExecutionInProgress
		} else {
			break
		}

		stepsLeft := len(ph.Steps)
		// --- 2. Iterate over phase steps ---
		for _, st := range ph.Steps {
			stepStatus := getOrCreateStepStatus(st.Name, phaseStatus)

			// Check current phase status: skip if finished, proceed if in progress, break out if a fatal error has occurred
			if isFinished(stepStatus.Status) {
				stepsLeft = stepsLeft - 1
				continue
			} else if isInProgress(stepStatus.Status) {
				stepStatus.Status = v1alpha1.ExecutionInProgress
			} else {
				// we are not in progress and not finished. An unexpected error occurred so that we can not proceed to the next phase
				break
			}

			tasksLeft := len(st.Tasks)
			// --- 3. Iterate over step tasks ---
			for _, tn := range st.Tasks {
				t, ok := pl.taskByName(tn)
				if !ok {
					phaseStatus.Status = v1alpha1.ExecutionFatalError
					stepStatus.Status = v1alpha1.ExecutionFatalError
					planStatus.Status = v1alpha1.ExecutionFatalError
					return planStatus, ExecutionError{
						Err:       fmt.Errorf("failed to find task %s for operator version %s", t, em.OperatorVersionName),
						Fatal:     true,
						EventName: &UnknownTaskNameEventName,
					}
				}
				// - 3.a build execution metadata -
				exm := ExecutionMetadata{
					EngineMetadata: *em,
					PlanName:       pl.name,
					PhaseName:      ph.Name,
					StepName:       st.Name,
					TaskName:       tn,
				}

				// - 3.b build the engine task -
				task, err := engtask.Build(t)
				if err != nil {
					phaseStatus.Status = v1alpha1.ExecutionFatalError
					stepStatus.Status = v1alpha1.ExecutionFatalError
					planStatus.Status = v1alpha1.ExecutionFatalError
					return planStatus, ExecutionError{
						Err:       fmt.Errorf("failed to resolve task %s for operator version %s: %w", tn, em.OperatorVersionName, err),
						Fatal:     true,
						EventName: &UnknownTaskKindEventName,
					}
				}

				// - 3.c build task context -
				ctx := engtask.Context{
					Client:     c,
					Enhancer:   enh,
					Meta:       exm,
					Templates:  pl.templates,
					Parameters: pl.params,
				}

				// --- 4. Execute the engine task ---
				done, err := task.Run(ctx)

				// Tasks within a step are executed "in parallel", meaning that we will not wait for current task to
				// be ready before executing the next one. However, this does not apply to fatal errors! Should a
				// fatal error occur, we will, in the spirit of "fail-loud-and-proud", abort current execution.
				//
				// Note: a task fatal error is set all the way through Plan/Phase/Step status fields:
				// PlanA: FATAL_ERROR
				//   PhaseB: FATAL_ERROR
				//     StepOne: FATAL_ERROR
				//
				// This is different for transient errors. Transient errors are retryable, so the corresponding Plan/Phase
				// are still "in progress":
				// PlanA: IN_PROGRESS
				//   PhaseB: IN_PROGRESS
				//     StepOne: ERROR
				switch {
				case errors.Is(err, engtask.FatalExecutionError):
					log.Printf("PlanExecution: fatal error during task %s execution for operator version %s: %v", exm.TaskName, exm.OperatorVersionName, err)
					phaseStatus.Status = v1alpha1.ExecutionFatalError
					stepStatus.Status = v1alpha1.ExecutionFatalError
					planStatus.Status = v1alpha1.ExecutionFatalError
					return planStatus, ExecutionError{
						Err:       fmt.Errorf("fatal task %s execution error  for operator version %s: %w", tn, em.OperatorVersionName, err),
						Fatal:     true,
						EventName: &FatalTaskExecutionErrorEventName,
					}
				case err != nil:
					log.Printf("PlanExecution: error during task %s execution for operator version %s: %v", exm.TaskName, exm.OperatorVersionName, err)
					stepStatus.Status = v1alpha1.ErrorStatus
				case done:
					tasksLeft = tasksLeft - 1
				}
			}

			// if some TASKs aren't ready yet and STEPs strategy is serial we can not proceed.
			if tasksLeft > 0 {
				if ph.Strategy == v1alpha1.Serial {
					log.Printf("PlanExecution: some tasks of the %s.%s, operator version %s are not ready", ph.Name, st.Name, em.OperatorVersionName)
					break
				}
			} else {
				stepStatus.Status = v1alpha1.ExecutionComplete
			}
			// otherwise, if STEPs strategy is parallel or all TASKs are finished, we can go to the next STEP:
			stepsLeft = stepsLeft - 1
		}

		// if some STEPs aren't ready yet and PHASEs strategy is serial we can not proceed.
		if stepsLeft > 0 {
			if pl.spec.Strategy == v1alpha1.Serial {
				log.Printf("PlanExecution: some steps of the %s.%s, operator version %s are not ready", pl.Name, ph.Name, em.OperatorVersionName)
				break
			}
		} else {
			phaseStatus.Status = v1alpha1.ExecutionComplete
		}
		// otherwise, if PHASEs strategy is parallel or all STEPs are finished, we can go to the next PHASE:
		phasesLeft = phasesLeft - 1
	}

	if phasesLeft == 0 {
		log.Printf("PlanExecution: All phases on plan %s and instance %s are healthy", pl.name, em.InstanceName)
		planStatus.Status = v1alpha1.ExecutionComplete
	}

	return planStatus, nil
}

// ======================================

// executePlan takes a currently active plan and ExecutionMetadata from the underlying operator and executes next "step" in that execution
// the next step could consist of actually executing multiple steps of the plan or just one depending on the execution strategy of the phase (serial/parallel)
// result of running this function is new state of the execution that is returned to the caller (it can either be completed, or still in progress or errored)
// in case of error, error is returned along with the state as well (so that it's possible to report which step caused the error)
// in case of error, method returns ErrorStatus which has property to indicate unrecoverable error meaning if there is no point in retrying that execution
func executePlan(plan *activePlan, metadata *EngineMetadata, c client.Client, enhancer KubernetesObjectEnhancer) (*v1alpha1.PlanStatus, error) {
	if plan.Status.IsTerminal() {
		log.Printf("PlanExecution: Plan %s for instance %s is terminal, nothing to do", plan.name, metadata.InstanceName)
		return plan.PlanStatus, nil
	}

	// we don't want to modify the original state, and State does not contain any pointer, so shallow copy is enough
	newState := &(*plan.PlanStatus)

	// render kubernetes resources needed to execute this plan
	planResources, err := prepareKubeResources(plan, metadata, enhancer)
	if err != nil {
		var exErr *ExecutionError
		if errors.As(err, &exErr) {
			newState.Status = v1alpha1.ExecutionFatalError
		} else {
			newState.Status = v1alpha1.ErrorStatus
		}
		return newState, err
	}

	// do a next step in the current plan execution
	allPhasesCompleted := true
	for _, ph := range plan.spec.Phases {
		currentPhaseState := getOrCreatePhaseStatus(ph.Name, newState)
		if isFinished(currentPhaseState.Status) {
			// nothing to do
			log.Printf("PlanExecution: Phase %s on plan %s and instance %s is in state %s, nothing to do", ph.Name, plan.name, metadata.InstanceName, currentPhaseState.Status)
			continue
		} else if isInProgress(currentPhaseState.Status) {
			newState.Status = v1alpha1.ExecutionInProgress
			currentPhaseState.Status = v1alpha1.ExecutionInProgress
			log.Printf("PlanExecution: Executing phase %s on plan %s and instance %s - it's in progress", ph.Name, plan.name, metadata.InstanceName)

			// we're currently executing this phase
			allStepsHealthy := true
			for _, st := range ph.Steps {
				currentStepState := getOrCreateStepStatus(st.Name, currentPhaseState)
				resources := planResources.PhaseResources[ph.Name].StepResources[st.Name]

				log.Printf("PlanExecution: Executing step %s on plan %s and instance %s - it's in %s state", st.Name, plan.name, metadata.InstanceName, currentStepState.Status)
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
				log.Printf("PlanExecution: All steps on phase %s plan %s and instance %s are healthy", ph.Name, plan.name, metadata.InstanceName)
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
		log.Printf("PlanExecution: All phases on plan %s and instance %s are healthy", plan.name, metadata.InstanceName)
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
	err := c.Patch(context.TODO(), existingResource, client.ConstantPatch(types.StrategicMergePatchType, newResourceJSON))
	if err != nil {
		// Right now applying a Strategic Merge Patch to custom resources does not work. There is
		// certain Metadata needed, which when missing, leads to an invalid Content-Type Header and
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
// it also takes care of applying KUDO specific conventions to the resources like common labels
func prepareKubeResources(plan *activePlan, meta *EngineMetadata, renderer KubernetesObjectEnhancer) (*planResources, error) {
	configs := make(map[string]interface{})
	configs["OperatorName"] = meta.OperatorName
	configs["Name"] = meta.InstanceName
	configs["Namespace"] = meta.InstanceNamespace
	configs["Params"] = plan.params

	result := &planResources{
		PhaseResources: make(map[string]phaseResources),
	}

	for _, phase := range plan.spec.Phases {
		phaseState := getOrCreatePhaseStatus(phase.Name, plan.PlanStatus)
		perStepResources := make(map[string][]runtime.Object)
		result.PhaseResources[phase.Name] = phaseResources{
			StepResources: perStepResources,
		}
		for j, step := range phase.Steps {
			configs["PlanName"] = plan.name
			configs["PhaseName"] = phase.Name
			configs["StepName"] = step.Name
			configs["StepNumber"] = strconv.FormatInt(int64(j), 10)
			var resources []runtime.Object
			stepState := getOrCreateStepStatus(step.Name, phaseState)

			engine := kudoengine.New()
			for _, tn := range step.Tasks {
				if task, ok := plan.taskByName(tn); ok {
					resourcesAsString := make(map[string]string)

					for _, res := range task.Spec.Resources {
						if resource, ok := plan.templates[res]; ok {
							templatedYaml, err := engine.Render(resource, configs)
							if err != nil {
								phaseState.Status = v1alpha1.ExecutionFatalError
								stepState.Status = v1alpha1.ExecutionFatalError

								err := errwrap.Wrap(err, "error expanding template")
								log.Print(err)
								return nil, &ExecutionError{err, true, nil}
							}
							resourcesAsString[res] = templatedYaml
						} else {
							phaseState.Status = v1alpha1.ExecutionFatalError
							stepState.Status = v1alpha1.ExecutionFatalError

							err := fmt.Errorf("PlanExecution: Error finding resource named %v for operator version %v", res, meta.OperatorVersionName)
							log.Print(err)
							return nil, &ExecutionError{err, true, nil}
						}
					}

					resourcesWithConventions, err := renderer.ApplyConventionsToTemplates(resourcesAsString, ExecutionMetadata{
						EngineMetadata: *meta,
						PlanName:       plan.name,
						PhaseName:      phase.Name,
						StepName:       step.Name,
					})

					if err != nil {
						phaseState.Status = v1alpha1.ErrorStatus
						stepState.Status = v1alpha1.ErrorStatus

						log.Printf("Error creating Kubernetes objects from step %v in phase %v of plan %v and instance %s/%s: %v", step.Name, phase.Name, plan.name, meta.InstanceNamespace, meta.InstanceName, err)
						return nil, &ExecutionError{err, false, nil}
					}
					resources = append(resources, resourcesWithConventions...)
				} else {
					phaseState.Status = v1alpha1.ErrorStatus
					stepState.Status = v1alpha1.ErrorStatus

					err := fmt.Errorf("Error finding task named %s for operator version %s", task, meta.OperatorVersionName)
					log.Print(err)
					return nil, &ExecutionError{err, false, nil}
				}
			}

			perStepResources[step.Name] = resources
		}
	}

	return result, nil
}

func getOrCreateStepStatus(stepName string, phaseStatus *v1alpha1.PhaseStatus) *v1alpha1.StepStatus {
	for i, p := range phaseStatus.Steps {
		if p.Name == stepName {
			return &phaseStatus.Steps[i]
		}
	}

	log.Printf("PlanExecution: creating missing (!) step %s status for the phase status: %+v", stepName, phaseStatus)
	stepStatus := &v1alpha1.StepStatus{Name: stepName}
	phaseStatus.Steps = append(phaseStatus.Steps, *stepStatus)

	return stepStatus
}

func getOrCreatePhaseStatus(phaseName string, planStatus *v1alpha1.PlanStatus) *v1alpha1.PhaseStatus {
	for i, p := range planStatus.Phases {
		if p.Name == phaseName {
			return &planStatus.Phases[i]
		}
	}

	log.Printf("PlanExecution: creating missing (!) phase %s status in plan status: %+v", phaseName, planStatus)
	phaseStatus := &v1alpha1.PhaseStatus{Name: phaseName}
	planStatus.Phases = append(planStatus.Phases, *phaseStatus)
	return phaseStatus
}

func isFinished(state v1alpha1.ExecutionStatus) bool {
	return state == v1alpha1.ExecutionComplete
}

func isInProgress(state v1alpha1.ExecutionStatus) bool {
	return state == v1alpha1.ExecutionInProgress || state == v1alpha1.ExecutionPending || state == v1alpha1.ErrorStatus
}
