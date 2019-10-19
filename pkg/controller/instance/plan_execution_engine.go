package instance

import (
	"errors"
	"fmt"
	"log"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	engtask "github.com/kudobuilder/kudo/pkg/engine/task"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	unknownTaskNameEventName         = "UnknownTaskName"
	unknownTaskKindEventName         = "UnknownTaskKind"
	fatalTaskExecutionErrorEventName = "FatalTaskExecutionError"
	missingPhaseStatus               = "MissingPhaseStatus"
	missingStepStatus                = "MissingStepStatus"
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

// executePlan method takes a currently active plan and ExecutionMetadata from the underlying operator and executes it.
// An execution loop iterates through plan phases, steps and tasks, executing them according to the execution strategy
// (serial/parallel). Task execution might result in success, error and fatal error. It is to distinguish between transient
// and fatal errors.  Transient errors are retryable, so the corresponding Plan/Phase are still in progress:
//  └── first-operator-zljnmj
//     └── Plan deploy (serial strategy) [IN_PROGRESS]
//        └── Phase main [IN_PROGRESS]
//           └── Step everything (ERROR)
//
// However, this does not apply to fatal errors! Should a  fatal error occur, we will, in the spirit of "fail-loud-and-proud",
// abort current execution, resulting in a plan status like:
//  └── first-operator-zljnmj
//     └── Plan deploy (serial strategy) [FATAL_ERROR]
//        └── Phase main [FATAL_ERROR]
//           └── Step everything (FATAL_ERROR)
//
// Furthermore, a transient ERROR during a step execution, means that the next step may be executed if the step strategy
// is "parallel". In case of a fatal error, it is returned alongside with the new plan status and published on the event bus.
func executePlan(pl *activePlan, em *engtask.EngineMetadata, c client.Client, enh engtask.KubernetesObjectEnhancer) (*v1alpha1.PlanStatus, error) {
	if pl.Status.IsTerminal() {
		log.Printf("PlanExecution: Plan %s for instance %s is terminal, nothing to do", pl.name, em.InstanceName)
		return pl.PlanStatus, nil
	}

	planStatus := pl.PlanStatus.DeepCopy()
	planStatus.Status = v1alpha1.ExecutionInProgress

	phasesLeft := len(pl.spec.Phases)
	// --- 1. Iterate over plan phases ---
	for _, ph := range pl.spec.Phases {
		phaseStatus := getPhaseStatus(ph.Name, planStatus)
		if phaseStatus == nil {
			planStatus.Status = v1alpha1.ExecutionFatalError
			return planStatus, ExecutionError{
				Err:       fmt.Errorf("failed to find phase %s for operator version %s", ph.Name, em.OperatorVersionName),
				Fatal:     true,
				EventName: &missingPhaseStatus,
			}
		}

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
			stepStatus := getStepStatus(st.Name, phaseStatus)
			if stepStatus == nil {
				phaseStatus.Status = v1alpha1.ExecutionFatalError
				planStatus.Status = v1alpha1.ExecutionFatalError
				return planStatus, ExecutionError{
					Err:       fmt.Errorf("failed to find step %s for operator version %s", st.Name, em.OperatorVersionName),
					Fatal:     true,
					EventName: &missingStepStatus,
				}
			}

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
						Err:       fmt.Errorf("failed to find task %s for operator version %s", tn, em.OperatorVersionName),
						Fatal:     true,
						EventName: &unknownTaskNameEventName,
					}
				}
				// - 3.a build execution metadata -
				exm := engtask.ExecutionMetadata{
					EngineMetadata: *em,
					PlanName:       pl.name,
					PhaseName:      ph.Name,
					StepName:       st.Name,
					TaskName:       tn,
				}

				// - 3.b build the engine task -
				task, err := engtask.Build(t)
				if err != nil {
					stepStatus.Status = v1alpha1.ExecutionFatalError
					phaseStatus.Status = v1alpha1.ExecutionFatalError
					planStatus.Status = v1alpha1.ExecutionFatalError
					return planStatus, ExecutionError{
						Err:       fmt.Errorf("failed to resolve task %s for operator version %s: %w", tn, em.OperatorVersionName, err),
						Fatal:     true,
						EventName: &unknownTaskKindEventName,
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

				// a fatal error is propagated through the plan/phase/step statuses and the plan execution will be
				// stopped in the spirit of "fail-loud-and-proud".
				switch {
				case errors.Is(err, engtask.ErrFatalExecution):
					log.Printf("PlanExecution: error during task %s execution for operator version %s: %v", exm.TaskName, exm.OperatorVersionName, err)
					phaseStatus.Status = v1alpha1.ExecutionFatalError
					stepStatus.Status = v1alpha1.ExecutionFatalError
					planStatus.Status = v1alpha1.ExecutionFatalError
					return planStatus, ExecutionError{
						Err:       fmt.Errorf("error during task %s execution for operator version %s: %w", tn, em.OperatorVersionName, err),
						Fatal:     true,
						EventName: &fatalTaskExecutionErrorEventName,
					}
				case err != nil:
					log.Printf("PlanExecution: error during task %s execution for operator version %s: %v", exm.TaskName, exm.OperatorVersionName, err)
					stepStatus.Status = v1alpha1.ErrorStatus
				case done:
					tasksLeft = tasksLeft - 1
				}
			}

			// --- 5. Check if all TASKs are finished ---
			// if some TASKs aren't ready yet and STEPs strategy is serial we can not proceed
			// otherwise, if STEPs strategy is parallel or all TASKs are finished, we can go to the next STEP
			if tasksLeft > 0 {
				if ph.Strategy == v1alpha1.Serial {
					log.Printf("PlanExecution: some tasks of the %s.%s, operator version %s are not ready", ph.Name, st.Name, em.OperatorVersionName)
					break
				}
			} else {
				stepStatus.Status = v1alpha1.ExecutionComplete
				stepsLeft = stepsLeft - 1
			}
		}

		// --- 6. Check if all STEPs are finished ---
		// if some STEPs aren't ready yet and PHASEs strategy is serial we can not proceed
		// otherwise, if PHASEs strategy is parallel or all STEPs are finished, we can go to the next PHASE
		if stepsLeft > 0 {
			if pl.spec.Strategy == v1alpha1.Serial {
				log.Printf("PlanExecution: some steps of the %s.%s, operator version %s are not ready", pl.Name, ph.Name, em.OperatorVersionName)
				break
			}
		} else {
			phaseStatus.Status = v1alpha1.ExecutionComplete
			phasesLeft = phasesLeft - 1
		}
	}

	// --- 7. Check if all PHASEs are finished ---
	if phasesLeft == 0 {
		log.Printf("PlanExecution: All phases on plan %s and instance %s are healthy", pl.name, em.InstanceName)
		planStatus.Status = v1alpha1.ExecutionComplete
	}

	return planStatus, nil
}

func getStepStatus(stepName string, phaseStatus *v1alpha1.PhaseStatus) *v1alpha1.StepStatus {
	for i, p := range phaseStatus.Steps {
		if p.Name == stepName {
			return &phaseStatus.Steps[i]
		}
	}

	return nil
}

func getPhaseStatus(phaseName string, planStatus *v1alpha1.PlanStatus) *v1alpha1.PhaseStatus {
	for i, p := range planStatus.Phases {
		if p.Name == phaseName {
			return &planStatus.Phases[i]
		}
	}

	return nil
}

func isFinished(state v1alpha1.ExecutionStatus) bool {
	return state == v1alpha1.ExecutionComplete
}

func isInProgress(state v1alpha1.ExecutionStatus) bool {
	return state == v1alpha1.ExecutionInProgress || state == v1alpha1.ExecutionPending || state == v1alpha1.ErrorStatus
}
