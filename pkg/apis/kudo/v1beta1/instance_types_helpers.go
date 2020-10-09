package v1beta1

import (
	"context"
	"fmt"
	"log"

	"github.com/thoas/go-funk"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	instanceCleanupFinalizerName = "kudo.dev.instance.cleanup"
)

func GetInstance(namespacedName types.NamespacedName, c client.Client) (i *Instance, err error) {
	i = &Instance{}
	err = c.Get(context.TODO(), namespacedName, i)
	if err != nil {
		return nil, err
	}
	return i, nil
}

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
// also updates Ready condition for finished plans
func (i *Instance) UpdateInstanceStatus(ps *PlanStatus, updatedTimestamp *metav1.Time) {
	for k, v := range i.Status.PlanStatus {
		if v.Name == ps.Name {
			ps.LastUpdatedTimestamp = updatedTimestamp
			i.Status.PlanStatus[k] = *ps
			i.Spec.PlanExecution.Status = ps.Status
		}
	}
	if i.Spec.PlanExecution.Status.IsFinished() {
		i.SetReadiness(ReadinessResourcesReady, "")
	}
	if i.Spec.PlanExecution.Status == ExecutionFatalError {
		i.SetReadiness(ReadinessPlanInFatalError, "")
	}
}

// ResetPlanStatus method resets a PlanStatus for a passed plan name and instance. Plan/phase/step statuses
// are set to ExecutionPending meaning that the controller will restart plan execution.
func (i *Instance) ResetPlanStatus(ps *PlanStatus, uid types.UID, updatedTimestamp *metav1.Time) {
	ps.UID = uid
	ps.Status = ExecutionPending
	for i := range ps.Phases {
		ps.Phases[i].Set(ExecutionPending)

		for j := range ps.Phases[i].Steps {
			ps.Phases[i].Steps[j].Set(ExecutionPending)
		}
	}

	// update plan status and instance aggregated status
	i.UpdateInstanceStatus(ps, updatedTimestamp)
}

// IsDeleting returns true is the instance is being deleted.
func (i *Instance) IsDeleting() bool {
	// a delete request is indicated by a non-zero 'metadata.deletionTimestamp',
	// see https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#finalizers
	return !i.ObjectMeta.DeletionTimestamp.IsZero()
}

func (i *Instance) HasNoFinalizers() bool { return len(i.GetFinalizers()) == 0 }

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

func (i *Instance) HasCleanupFinalizer() bool {
	return funk.ContainsString(i.ObjectMeta.Finalizers, instanceCleanupFinalizerName)
}

// TryAddFinalizer adds the cleanup finalizer to an instance if the finalizer
// hasn't been added yet, the instance has a cleanup plan and the cleanup plan
// didn't run yet. Returns true if the cleanup finalizer has been added.
func (i *Instance) TryAddFinalizer() bool {
	if !i.HasCleanupFinalizer() {
		planStatus := i.PlanStatus(CleanupPlanName)
		// avoid adding a finalizer multiple times: we only add it if the corresponding
		// plan.Status is nil (meaning the plan never ran) or if it exists but equals ExecutionNeverRun
		if planStatus == nil || planStatus.Status == ExecutionNeverRun {
			i.ObjectMeta.Finalizers = append(i.ObjectMeta.Finalizers, instanceCleanupFinalizerName)
			return true
		}
	}

	return false
}

// TryRemoveFinalizer removes the cleanup finalizer of an instance if it has
// been added, the instance has a cleanup plan and the cleanup plan *successfully* finished.
// Returns true if the cleanup finalizer has been removed.
func (i *Instance) TryRemoveFinalizer() bool {
	if funk.ContainsString(i.ObjectMeta.Finalizers, instanceCleanupFinalizerName) {
		if planStatus := i.PlanStatus(CleanupPlanName); planStatus != nil {
			// we check IsFinished and *not* IsTerminal here so that the finalizer is not removed in the FatalError
			// case. This way a human operator has to intervene and we don't leave garbage in the cluster.
			if planStatus.Status.IsFinished() {
				log.Printf("Removing finalizer on instance %s/%s, cleanup plan is finished", i.Namespace, i.Name)
				i.ObjectMeta.Finalizers = remove(i.ObjectMeta.Finalizers, instanceCleanupFinalizerName)
				return true
			}
		} else {
			// We have a finalizer but no cleanup plan. This could be due to an updated instance.
			// Let's remove the finalizer.
			log.Printf("Removing finalizer on instance %s/%s because there is no cleanup plan", i.Namespace, i.Name)
			i.ObjectMeta.Finalizers = remove(i.ObjectMeta.Finalizers, instanceCleanupFinalizerName)
			return true
		}
	}

	return false
}

func remove(values []string, s string) []string {
	return funk.FilterString(values, func(str string) bool {
		return str != s
	})
}

// GetOperatorVersion retrieves OperatorVersion belonging to the given instance
func (i *Instance) GetOperatorVersion(c client.Reader) (ov *OperatorVersion, err error) {
	return GetOperatorVersionByName(i.Spec.OperatorVersion.Name, i.OperatorVersionNamespace(), c)
}

// IsChildInstance method return true if this instance is owned by another instance (as a dependency) and false otherwise.
// If there is any owner with the same kind 'Instance' then this Instance is owned by another one.
func (i *Instance) IsChildInstance() bool {
	for _, or := range i.GetOwnerReferences() {
		if or.Kind == i.Kind {
			return true
		}
	}
	return false
}

func (i *Instance) IsTopLevelInstance() bool {
	return !i.IsChildInstance()
}

type ReadinessType string

const (
	ReadinessPlanInProgress   ReadinessType = "PlanInProgress"
	ReadinessPlanInFatalError ReadinessType = "PlanInFatalError"
	ReadinessResourceNotReady ReadinessType = "ResourceNotReady"
	ReadinessResourcesReady   ReadinessType = "ResourcesReady"

	readyConditionType = "Ready"
)

func (i *Instance) SetReadiness(reason ReadinessType, msg string) {
	var status metav1.ConditionStatus

	switch reason {
	case ReadinessResourcesReady:
		status = metav1.ConditionTrue
	case ReadinessResourceNotReady:
		status = metav1.ConditionFalse
	case ReadinessPlanInFatalError, ReadinessPlanInProgress:
		status = metav1.ConditionUnknown
	}

	condition := metav1.Condition{Type: readyConditionType, Status: status, Message: msg, Reason: string(reason)}
	meta.SetStatusCondition(&i.Status.Conditions, condition)
}

// wasRunAfter returns true if p1 was run after p2
func wasRunAfter(p1 PlanStatus, p2 PlanStatus) bool {
	if p1.Status == ExecutionNeverRun || p2.Status == ExecutionNeverRun || p1.LastUpdatedTimestamp == nil || p2.LastUpdatedTimestamp == nil {
		return false
	}
	return p1.LastUpdatedTimestamp.Time.After(p2.LastUpdatedTimestamp.Time)
}

// GetParamDefinitions retrieves parameter metadata from OperatorVersion but returns an error if any parameter is missing
func GetParamDefinitions(params map[string]string, ov *OperatorVersion) ([]Parameter, error) {
	defs := []Parameter{}
	for p1 := range params {
		p1 := p1
		p2 := funk.Find(ov.Spec.Parameters, func(e Parameter) bool {
			return e.Name == p1
		})

		if p2 == nil {
			return nil, fmt.Errorf("failed to find parameter %q in the OperatorVersion", p1)
		}

		defs = append(defs, p2.(Parameter))
	}
	return defs, nil
}

func CleanupPlanExists(ov *OperatorVersion) bool { return PlanExists(CleanupPlanName, ov) }

func PlanExists(plan string, ov *OperatorVersion) bool {
	_, ok := ov.Spec.Plans[plan]
	return ok
}

// SelectPlan returns nil if none of the plan exists, otherwise the first one in list that exists
func SelectPlan(possiblePlans []string, ov *OperatorVersion) *string {
	for _, plan := range possiblePlans {
		if _, ok := ov.Spec.Plans[plan]; ok {
			return &plan
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
