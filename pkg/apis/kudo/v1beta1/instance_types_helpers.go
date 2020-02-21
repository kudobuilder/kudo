package v1beta1

import (
	"fmt"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

// GetPlanInProgress returns plan status of currently active plan or nil if no plan is running
func (i *Instance) GetPlanInProgress() *PlanStatus {
	for _, p := range i.Status.PlanStatus {
		if p.Status.IsRunning() {
			return &p
		}
	}
	return nil
}

// GetLastExecutedPlanStatus returns status of plan that is currently running, if there is one running
// if no plan is running it looks for last executed plan based on timestamps
func (i *Instance) GetLastExecutedPlanStatus() *PlanStatus {
	if i.NoPlanEverExecuted() {
		return nil
	}
	activePlan := i.GetPlanInProgress()
	if activePlan != nil {
		return activePlan
	}
	var lastExecutedPlan *PlanStatus
	for n := range i.Status.PlanStatus {
		p := i.Status.PlanStatus[n]
		if p.Status == ExecutionNeverRun {
			continue // only interested in plans that run
		}
		if lastExecutedPlan == nil {
			lastExecutedPlan = &p // first plan that was run and we're iterating over
		} else if wasRunAfter(p, *lastExecutedPlan) {
			lastExecutedPlan = &p // this plan was run after the plan we have chosen before
		}
	}
	return lastExecutedPlan
}

// NoPlanEverExecuted returns true is this is new instance for which we never executed any plan
func (i *Instance) NoPlanEverExecuted() bool {
	for _, p := range i.Status.PlanStatus {
		if p.Status != ExecutionNeverRun {
			return false
		}
	}
	return true
}

// UpdateInstanceStatus updates `Status.PlanStatus` and `Status.AggregatedStatus` property based on the given plan
func (i *Instance) UpdateInstanceStatus(planStatus *PlanStatus) {
	currentTime := time.Now()
	for k, v := range i.Status.PlanStatus {
		if v.Name == planStatus.Name {
			i.Status.PlanStatus[k] = *planStatus
			i.Status.AggregatedStatus.Status = planStatus.Status
			i.Status.AggregatedStatus.LastUpdated = &v1.Time{Time: currentTime}
			if planStatus.Status.IsTerminal() {
				i.Status.AggregatedStatus.ActivePlanName = ""
			}
		}
	}
}

// ResetPlanStatus method resets a PlanStatus for a passed plan name and instance. Plan/phase/step statuses
// are set to ExecutionPending meaning that the controller will restart plan execution.
func (i *Instance) ResetPlanStatus(plan string) error {
	planStatus := i.PlanStatus(plan)
	if planStatus == nil {
		return fmt.Errorf("failed to find planStatus for the plan '%s'", plan)
	}

	// reset plan's phases and steps by setting them to ExecutionPending
	planStatus.Set(ExecutionPending)
	planStatus.UID = uuid.NewUUID()

	for i, ph := range planStatus.Phases {
		planStatus.Phases[i].Set(ExecutionPending)

		for j := range ph.Steps {
			planStatus.Phases[i].Steps[j].Set(ExecutionPending)
		}
	}

	// update instance aggregated status
	i.UpdateInstanceStatus(planStatus)
	return nil
}

// IsDeleting returns true is the instance is being deleted.
func (i *Instance) IsDeleting() bool {
	// a delete request is indicated by a non-zero 'metadata.deletionTimestamp',
	// see https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#finalizers
	return !i.ObjectMeta.DeletionTimestamp.IsZero()
}

// OperatorVersionNamespace returns the namespace of the OperatorVersion that the Instance references.
func (i *Instance) OperatorVersionNamespace() string {
	if i.Spec.OperatorVersion.Namespace == "" {
		return i.ObjectMeta.Namespace
	}
	return i.Spec.OperatorVersion.Namespace
}

func (i *Instance) PlanStatus(plan string) *PlanStatus {
	for _, planStatus := range i.Status.PlanStatus {
		if planStatus.Name == plan {
			return &planStatus
		}
	}

	return nil
}

// wasRunAfter returns true if p1 was run after p2
func wasRunAfter(p1 PlanStatus, p2 PlanStatus) bool {
	if p1.Status == ExecutionNeverRun || p2.Status == ExecutionNeverRun || p1.LastFinishedRun == nil || p2.LastFinishedRun == nil {
		return false
	}
	return p1.LastFinishedRun.Time.After(p2.LastFinishedRun.Time)
}

// GetParamDefinitions retrieves parameter metadata from OperatorVersion CRD
func GetParamDefinitions(params map[string]string, ov *OperatorVersion) []Parameter {
	defs := []Parameter{}
	for p1 := range params {
		for _, p2 := range ov.Spec.Parameters {
			if p2.Name == p1 {
				defs = append(defs, p2)
			}
		}
	}
	return defs
}

// ParameterDiff returns map containing all parameters that were removed or changed between old and new
func ParameterDiff(old, new map[string]string) map[string]string {
	diff := make(map[string]string)

	for key, val := range old {
		// If a parameter was removed in the new spec
		if _, ok := new[key]; !ok {
			diff[key] = val
		}
	}

	for key, val := range new {
		// If new spec parameter was added or changed
		if v, ok := old[key]; !ok || v != val {
			diff[key] = val
		}
	}

	return diff
}

// SelectPlan returns nil if none of the plan exists, otherwise the first one in list that exists
func SelectPlan(possiblePlans []string, ov *OperatorVersion) *string {
	for _, n := range possiblePlans {
		if _, ok := ov.Spec.Plans[n]; ok {
			return kudo.String(n)
		}
	}
	return nil
}

func GetStepStatus(stepName string, phaseStatus *PhaseStatus) *StepStatus {
	for i, p := range phaseStatus.Steps {
		if p.Name == stepName {
			return &phaseStatus.Steps[i]
		}
	}

	return nil
}

func GetPhaseStatus(phaseName string, planStatus *PlanStatus) *PhaseStatus {
	for i, p := range planStatus.Phases {
		if p.Name == phaseName {
			return &planStatus.Phases[i]
		}
	}

	return nil
}
