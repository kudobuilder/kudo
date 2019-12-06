package workflow

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kudobuilder/kudo/pkg/engine/renderer"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/engine/task"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	unknownTaskNameEventName = "UnknownTaskName"
	unknownTaskKindEventName = "UnknownTaskKind"
	missingPhaseStatus       = "MissingPhaseStatus"
	missingStepStatus        = "MissingStepStatus"
)

// ActivePlan wraps over all data that is needed for its execution including tasks, templates, parameters etc.
type ActivePlan struct {
	Name string
	*v1beta1.PlanStatus
	Spec      *v1beta1.Plan
	Tasks     []v1beta1.Task
	Templates map[string]string
	Params    map[string]string
	Pipes     map[string]string
}

func (ap *ActivePlan) taskByName(name string) (*v1beta1.Task, bool) {
	for _, t := range ap.Tasks {
		if t.Name == name {
			return &t, true
		}
	}
	return nil, false
}

// Execute method takes a currently active plan and Metadata from the underlying operator and executes it.
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
// In terms of Status Message, we don't propagate the message up for fatal errors
//
// Furthermore, a transient ERROR during a step execution, means that the next step may be executed if the step strategy
// is "parallel". In case of a fatal error, it is returned alongside with the new plan status and published on the event bus.
func Execute(pl *ActivePlan, em *engine.Metadata, c client.Client, enh renderer.Enhancer, currentTime time.Time) (*v1beta1.PlanStatus, error) {
	if pl.Status.IsTerminal() {
		log.Printf("PlanExecution: %s/%s plan %s is terminal, nothing to do", em.InstanceNamespace, em.InstanceName, pl.Name)
		return pl.PlanStatus, nil
	}

	planStatus := pl.PlanStatus.DeepCopy()
	planStatus.Set(v1beta1.ExecutionInProgress)

	phasesLeft := len(pl.Spec.Phases)
	// --- 1. Iterate over plan phases ---
	for _, ph := range pl.Spec.Phases {
		phaseStatus := getPhaseStatus(ph.Name, planStatus)
		if phaseStatus == nil {
			err := fmt.Errorf("%s/%s %w missing phase status: %s.%s", em.InstanceNamespace, em.InstanceName, engine.ErrFatalExecution, pl.Name, ph.Name)

			planStatus.SetWithMessage(v1beta1.ExecutionFatalError, err.Error())
			return planStatus, engine.ExecutionError{
				Err:       err,
				EventName: missingPhaseStatus,
			}
		}

		// Check current phase status: skip if finished, proceed if in progress, break out if a fatal error has occurred
		if isFinished(phaseStatus.Status) {
			phasesLeft = phasesLeft - 1
			continue
		} else if isInProgress(phaseStatus.Status) {
			phaseStatus.Set(v1beta1.ExecutionInProgress)
		} else {
			break
		}

		stepsLeft := stepNamesToSet(ph.Steps)
		// --- 2. Iterate over phase steps ---
		for _, st := range ph.Steps {
			stepStatus := getStepStatus(st.Name, phaseStatus)
			if stepStatus == nil {
				err := fmt.Errorf("%s/%s %w missing step status: %s.%s.%s", em.InstanceNamespace, em.InstanceName, engine.ErrFatalExecution, pl.Name, ph.Name, st.Name)

				phaseStatus.SetWithMessage(v1beta1.ExecutionFatalError, err.Error())
				planStatus.Set(v1beta1.ExecutionFatalError)
				return planStatus, engine.ExecutionError{
					Err:       err,
					EventName: missingStepStatus,
				}
			}

			// Check current phase status: skip if finished, proceed if in progress, break out if a fatal error has occurred
			if isFinished(stepStatus.Status) {
				delete(stepsLeft, stepStatus.Name)
				continue
			} else if isInProgress(stepStatus.Status) {
				stepStatus.Set(v1beta1.ExecutionInProgress)
			} else {
				// we are not in progress and not finished. An unexpected error occurred so that we can not proceed to the next phase
				break
			}

			tasksLeft := stringArrayToSet(st.Tasks)
			// --- 3. Iterate over step tasks ---
			for _, tn := range st.Tasks {
				t, ok := pl.taskByName(tn)
				if !ok {
					err := fmt.Errorf("%s/%s %w missing task %s.%s.%s.%s", em.InstanceNamespace, em.InstanceName, engine.ErrFatalExecution, pl.Name, ph.Name, st.Name, tn)

					phaseStatus.Set(v1beta1.ExecutionFatalError)
					planStatus.Set(v1beta1.ExecutionFatalError)
					stepStatus.SetWithMessage(v1beta1.ExecutionFatalError, err.Error())
					return planStatus, engine.ExecutionError{
						Err:       err,
						EventName: unknownTaskNameEventName,
					}
				}
				// - 3.a build execution metadata -
				exm := renderer.Metadata{
					Metadata:  *em,
					PlanName:  pl.Name,
					PlanUID:   planStatus.UID,
					PhaseName: ph.Name,
					StepName:  st.Name,
					TaskName:  tn,
				}

				// - 3.b build the engine task -
				tt, err := task.Build(t)
				if err != nil {
					err := fmt.Errorf("%s/%s %w failed to build task %s.%s.%s.%s: %v", em.InstanceNamespace, em.InstanceName, engine.ErrFatalExecution, pl.Name, ph.Name, st.Name, tn, err)

					stepStatus.SetWithMessage(v1beta1.ExecutionFatalError, err.Error())
					planStatus.Set(v1beta1.ExecutionFatalError)
					phaseStatus.Set(v1beta1.ExecutionFatalError)
					return planStatus, engine.ExecutionError{
						Err:       err,
						EventName: unknownTaskKindEventName,
					}
				}

				// - 3.c build task context -
				ctx := task.Context{
					Client:     c,
					Enhancer:   enh,
					Meta:       exm,
					Templates:  pl.Templates,
					Parameters: pl.Params,
					Pipes:      pl.Pipes,
				}

				// --- 4. Execute the engine task ---
				done, err := tt.Run(ctx)

				// a fatal error is propagated through the plan/phase/step statuses and the plan execution will be
				// stopped in the spirit of "fail-loud-and-proud".
				switch {
				case errors.Is(err, engine.ErrFatalExecution):
					phaseStatus.Set(v1beta1.ExecutionFatalError)
					planStatus.Set(v1beta1.ExecutionFatalError)
					stepStatus.SetWithMessage(v1beta1.ExecutionFatalError, err.Error())
					return planStatus, err
				case err != nil:
					message := fmt.Sprintf("A transient error when executing task %s.%s.%s.%s. Will retry. %v", pl.Name, ph.Name, st.Name, t.Name, err)
					stepStatus.SetWithMessage(v1beta1.ErrorStatus, message)
					log.Printf("PlanExecution: %s", message)
				case done:
					delete(tasksLeft, t.Name)
				}
			}

			// --- 5. Check if all TASKs are finished ---
			// if some TASKs aren't ready yet and STEPs strategy is serial we can not proceed
			// otherwise, if STEPs strategy is parallel or all TASKs are finished, we can go to the next STEP
			if len(tasksLeft) > 0 {
				if ph.Strategy == v1beta1.Serial {
					log.Printf("PlanExecution: '%s' task(s) (instance: %s/%s) of the %s.%s.%s are not ready", mapKeysToString(tasksLeft), em.InstanceNamespace, em.InstanceName, pl.Name, ph.Name, st.Name)
					break
				}
			} else {
				stepStatus.Set(v1beta1.ExecutionComplete)
				delete(stepsLeft, stepStatus.Name)
			}
		}

		// --- 6. Check if all STEPs are finished ---
		// if some STEPs aren't ready yet and PHASEs strategy is serial we can not proceed
		// otherwise, if PHASEs strategy is parallel or all STEPs are finished, we can go to the next PHASE
		if len(stepsLeft) > 0 {
			if pl.Spec.Strategy == v1beta1.Serial {
				log.Printf("PlanExecution: '%s' step(s) (instance: %s/%s) of the %s.%s are not ready", mapKeysToString(stepsLeft), em.InstanceNamespace, em.InstanceName, pl.Name, ph.Name)
				break
			}
		} else {
			phaseStatus.Set(v1beta1.ExecutionComplete)
			phasesLeft = phasesLeft - 1
		}
	}

	// --- 7. Check if all PHASEs are finished ---
	if phasesLeft == 0 {
		log.Printf("PlanExecution: %s/%s all phases of the plan %s are ready", em.InstanceNamespace, em.InstanceName, pl.Name)
		planStatus.Set(v1beta1.ExecutionComplete)
		planStatus.LastFinishedRun = v1.Time{Time: currentTime}
	}

	return planStatus, nil
}

// mapKeysToString is helper method for getting map keys as comma separated string
func mapKeysToString(values map[string]bool) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}

	return strings.Join(keys, ",")
}

// stringArrayToSet converts slice of strings to map (set)
// this is useful to make it easier to remove from the collection
func stringArrayToSet(values []string) map[string]bool {
	set := make(map[string]bool)
	for _, t := range values {
		set[t] = true
	}
	return set
}

// stepNamesToSet converts slice of steps to map (set)
// this is useful to make it easier to remove from the collection and track what steps are finished
func stepNamesToSet(steps []v1beta1.Step) map[string]bool {
	set := make(map[string]bool)
	for _, s := range steps {
		set[s.Name] = true
	}
	return set
}

func getStepStatus(stepName string, phaseStatus *v1beta1.PhaseStatus) *v1beta1.StepStatus {
	for i, p := range phaseStatus.Steps {
		if p.Name == stepName {
			return &phaseStatus.Steps[i]
		}
	}

	return nil
}

func getPhaseStatus(phaseName string, planStatus *v1beta1.PlanStatus) *v1beta1.PhaseStatus {
	for i, p := range planStatus.Phases {
		if p.Name == phaseName {
			return &planStatus.Phases[i]
		}
	}

	return nil
}

func isFinished(state v1beta1.ExecutionStatus) bool {
	return state == v1beta1.ExecutionComplete
}

func isInProgress(state v1beta1.ExecutionStatus) bool {
	return state == v1beta1.ExecutionInProgress || state == v1beta1.ExecutionPending || state == v1beta1.ErrorStatus
}
