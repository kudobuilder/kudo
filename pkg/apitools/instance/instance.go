package instance

import "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"

// PlanStatus returns the status of the given plan in the instance, or nil if the instance does not have a plan with that name
func PlanStatus(in *v1beta1.Instance, plan string) *v1beta1.PlanStatus {
	for _, planStatus := range in.Status.PlanStatus {
		if planStatus.Name == plan {
			return &planStatus
		}
	}
	return nil
}

// UpdateInstanceStatus updates `Status.PlanStatus` and `Status.AggregatedStatus` property based on the given plan
func UpdateStatus(in *v1beta1.Instance, planStatus *v1beta1.PlanStatus) {
	for k, v := range in.Status.PlanStatus {
		if v.Name == planStatus.Name {
			in.Status.PlanStatus[k] = *planStatus
			in.Status.AggregatedStatus.Status = planStatus.Status
			if planStatus.Status.IsTerminal() {
				in.Status.AggregatedStatus.ActivePlanName = ""
			}
		}
	}
}

// OperatorVersionNamespace returns the namespace of the OperatorVersion that the Instance references.
func OperatorVersionNamespace(in *v1beta1.Instance) string {
	if in.Spec.OperatorVersion.Namespace == "" {
		return in.ObjectMeta.Namespace
	}
	return in.Spec.OperatorVersion.Namespace
}

func IsDeleting(in *v1beta1.Instance) bool {
	// a delete request is indicated by a non-zero 'metadata.deletionTimestamp',
	// see https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#finalizers
	return !in.ObjectMeta.DeletionTimestamp.IsZero()
}

// GetPlanInProgress returns plan status of currently active plan or nil if no plan is running
func GetPlanInProgress(in *v1beta1.Instance) *v1beta1.PlanStatus {
	for _, p := range in.Status.PlanStatus {
		if p.Status.IsRunning() {
			return &p
		}
	}
	return nil
}

// NoPlanEverExecuted returns true if in is a new instance for which never any plan was executed
func NoPlanEverExecuted(in *v1beta1.Instance) bool {
	for _, p := range in.Status.PlanStatus {
		if p.Status != v1beta1.ExecutionNeverRun {
			return false
		}
	}
	return true
}

// GetLastExecutedPlanStatus returns status of plan that is currently running, if there is one running
// if no plan is running it looks for last executed plan based on timestamps
func GetLastExecutedPlanStatus(in *v1beta1.Instance) *v1beta1.PlanStatus {
	if NoPlanEverExecuted(in) {
		return nil
	}
	activePlan := GetPlanInProgress(in)
	if activePlan != nil {
		return activePlan
	}
	var lastExecutedPlan *v1beta1.PlanStatus
	for n := range in.Status.PlanStatus {
		p := in.Status.PlanStatus[n]
		if p.Status == v1beta1.ExecutionNeverRun {
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

// wasRunAfter returns true if p1 was run after p2
func wasRunAfter(p1 v1beta1.PlanStatus, p2 v1beta1.PlanStatus) bool {
	if p1.Status == v1beta1.ExecutionNeverRun || p2.Status == v1beta1.ExecutionNeverRun {
		return false
	}
	return p1.LastFinishedRun.Time.After(p2.LastFinishedRun.Time)
}
