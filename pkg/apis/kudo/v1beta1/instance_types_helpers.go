package v1beta1

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
	for k, v := range i.Status.PlanStatus {
		if v.Name == planStatus.Name {
			i.Status.PlanStatus[k] = *planStatus
			i.Status.AggregatedStatus.Status = planStatus.Status
			if planStatus.Status.IsTerminal() {
				i.Status.AggregatedStatus.ActivePlanName = ""
			}
		}
	}
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
	if p1.Status == ExecutionNeverRun || p2.Status == ExecutionNeverRun {
		return false
	}
	return p1.LastFinishedRun.Time.After(p2.LastFinishedRun.Time)
}
