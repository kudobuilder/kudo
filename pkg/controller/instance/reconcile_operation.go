package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"

	"github.com/kudobuilder/kudo/pkg/util/kudo"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	kudov1beta1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"k8s.io/apimachinery/pkg/util/uuid"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const snapshotAnnotation = "kudo.dev/last-applied-instance-state"

type reconcileOperation struct {
	client      client.Client
	recorder    record.EventRecorder
	instance    *v1beta1.Instance
	oldInstance *v1beta1.Instance
	ov          *v1beta1.OperatorVersion
}

func newReconcileOperation(client client.Client, recorder record.EventRecorder) reconcileOperation {
	return reconcileOperation{
		client:   client,
		recorder: recorder,
	}
}

func (op reconcileOperation) load(req ctrl.Request) error {
	instance, err := op.getInstance(req)
	if err != nil {
		return err
	}
	ov, err := op.getOperatorVersion(instance)
	if err != nil {
		return err
	}

	op.instance = instance
	op.oldInstance = instance.DeepCopy()
	op.ov = ov
	return nil
}

func (op reconcileOperation) IsInstanceDeleting() bool {
	// a delete request is indicated by a non-zero 'metadata.deletionTimestamp',
	// see https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#finalizers
	return !op.instance.ObjectMeta.DeletionTimestamp.IsZero()
}

// GetPlanToBeExecuted returns name of the plan that should be executed
func (op reconcileOperation) GetPlanToBeExecuted() (*string, error) {
	if op.IsInstanceDeleting() {
		// we have a cleanup plan
		plan := selectPlan([]string{v1beta1.CleanupPlanName}, op.ov)
		if plan != nil {
			if planStatus := op.PlanStatus(*plan); planStatus != nil {
				if !planStatus.Status.IsRunning() {
					if planStatus.Status.IsFinished() {
						// we already finished the cleanup plan
						return nil, nil
					}
					return plan, nil
				}
			}
		}
	}

	if op.GetPlanInProgress() != nil { // we're already running some plan
		return nil, nil
	}

	// new instance, need to run deploy plan
	if op.NoPlanEverExecuted() {
		return kudo.String(v1beta1.DeployPlanName), nil
	}

	// did the instance change so that we need to run deploy/upgrade/update plan?
	instanceSnapshot, err := op.snapshotSpec()
	if err != nil {
		return nil, err
	}
	if instanceSnapshot == nil {
		// we don't have snapshot -> we never run deploy, also we cannot run update/upgrade. This should never happen
		return nil, &v1beta1.InstanceError{fmt.Errorf("unexpected state: no plan is running, no snapshot present - this should never happen :) for instance %s/%s", op.instance.Namespace, op.instance.Name), kudo.String("UnexpectedState")}
	}
	if instanceSnapshot.OperatorVersion.Name != op.instance.Spec.OperatorVersion.Name {
		// this instance was upgraded to newer version
		log.Printf("Instance: instance %s/%s was upgraded from %s to %s operatorVersion", op.instance.Namespace, op.instance.Name, instanceSnapshot.OperatorVersion.Name, op.instance.Spec.OperatorVersion.Name)
		plan := selectPlan([]string{v1beta1.UpgradePlanName, v1beta1.UpdatePlanName, v1beta1.DeployPlanName}, op.ov)
		if plan == nil {
			return nil, &v1beta1.InstanceError{fmt.Errorf("supposed to execute plan because instance %s/%s was upgraded but none of the deploy, upgrade, update plans found in linked operatorVersion", op.instance.Namespace, op.instance.Name), kudo.String("PlanNotFound")}
		}
		return plan, nil
	}
	// did instance parameters change, so that the corresponding plan has to be triggered?
	if !reflect.DeepEqual(instanceSnapshot.Parameters, op.instance.Spec.Parameters) {
		// instance updated
		log.Printf("Instance: instance %s/%s has updated parameters from %v to %v", op.instance.Namespace, op.instance.Name, instanceSnapshot.Parameters, op.instance.Spec.Parameters)
		paramDiff := parameterDifference(instanceSnapshot.Parameters, op.instance.Spec.Parameters)
		paramDefinitions := getParamDefinitions(paramDiff, op.ov)
		plan := planNameFromParameters(paramDefinitions, op.ov)
		if plan == nil {
			return nil, &v1beta1.InstanceError{fmt.Errorf("supposed to execute plan because instance %s/%s was updated but none of the deploy, update plans found in linked operatorVersion", op.instance.Namespace, op.instance.Name), kudo.String("PlanNotFound")}
		}
		return plan, nil
	}
	return nil, nil
}

// StartPlanExecution mark plan as to be executed
// this modifies the instance.Status as well as instance.Metadata.Annotation (to save snapshot if needed)
func (op reconcileOperation) StartPlanExecution(planName string) error {
	if op.NoPlanEverExecuted() || isUpgradePlan(planName) {
		op.EnsurePlanStatusInitialized()
	}

	// update status of the instance to reflect the newly starting plan
	notFound := true
	for planIndex, v := range op.instance.Status.PlanStatus {
		if v.Name == planName {
			// update plan status
			notFound = false
			planStatus := op.instance.Status.PlanStatus[planIndex]
			planStatus.Set(v1beta1.ExecutionPending)
			planStatus.UID = uuid.NewUUID()
			for j, p := range v.Phases {
				planStatus.Phases[j].Set(v1beta1.ExecutionPending)
				for k := range p.Steps {
					op.instance.Status.PlanStatus[planIndex].Phases[j].Steps[k].Set(v1beta1.ExecutionPending)
				}
			}

			op.instance.Status.PlanStatus[planIndex] = planStatus // we cannot modify item in map, we need to reassign here

			// update activePlan and instance status
			op.instance.Status.AggregatedStatus.Status = v1beta1.ExecutionPending
			op.instance.Status.AggregatedStatus.ActivePlanName = planName

			break
		}
	}
	if notFound {
		return &v1beta1.InstanceError{fmt.Errorf("asked to execute a plan %s but no such plan found in instance %s/%s", planName, op.instance.Namespace, op.instance.Name), kudo.String("PlanNotFound")}
	}

	err := op.SaveSnapshot()
	if err != nil {
		return err
	}

	return nil
}

// EnsurePlanStatusInitialized initializes plan status for all plans this instance supports
// it does not trigger run of any plan
// it either initializes everything for a fresh instance without any status or tries to adjust status after OV was updated
func (op reconcileOperation) EnsurePlanStatusInitialized() {
	if op.instance.Status.PlanStatus == nil {
		op.instance.Status.PlanStatus = make(map[string]v1beta1.PlanStatus)
	}

	for planName, plan := range op.ov.Spec.Plans {
		planStatus := &v1beta1.PlanStatus{
			Name:   planName,
			Status: v1beta1.ExecutionNeverRun,
			Phases: make([]v1beta1.PhaseStatus, 0),
		}

		existingPlanStatus, planExists := op.instance.Status.PlanStatus[planName]
		if planExists {
			planStatus.SetWithMessage(existingPlanStatus.Status, existingPlanStatus.Message)
		}
		for _, phase := range plan.Phases {
			phaseStatus := &v1beta1.PhaseStatus{
				Name:   phase.Name,
				Status: v1beta1.ExecutionNeverRun,
				Steps:  make([]v1beta1.StepStatus, 0),
			}
			existingPhaseStatus, phaseExists := v1beta1.PhaseStatus{}, false
			if planExists {
				for _, oldPhase := range existingPlanStatus.Phases {
					if phase.Name == oldPhase.Name {
						existingPhaseStatus = oldPhase
						phaseExists = true
						phaseStatus.SetWithMessage(existingPhaseStatus.Status, existingPhaseStatus.Message)
					}
				}
			}
			for _, step := range phase.Steps {
				stepStatus := v1beta1.StepStatus{
					Name:   step.Name,
					Status: v1beta1.ExecutionNeverRun,
				}
				if phaseExists {
					for _, oldStep := range existingPhaseStatus.Steps {
						if step.Name == oldStep.Name {
							stepStatus.SetWithMessage(oldStep.Status, oldStep.Message)
						}
					}
				}
				phaseStatus.Steps = append(phaseStatus.Steps, stepStatus)
			}
			planStatus.Phases = append(planStatus.Phases, *phaseStatus)
		}
		op.instance.Status.PlanStatus[planName] = *planStatus
	}
}

// UpdateInstanceStatus updates `Status.PlanStatus` and `Status.AggregatedStatus` property based on the given plan
func (op reconcileOperation) UpdateInstanceStatus(planStatus *v1beta1.PlanStatus) {
	for k, v := range op.instance.Status.PlanStatus {
		if v.Name == planStatus.Name {
			op.instance.Status.PlanStatus[k] = *planStatus
			op.instance.Status.AggregatedStatus.Status = planStatus.Status
			if planStatus.Status.IsTerminal() {
				op.instance.Status.AggregatedStatus.ActivePlanName = ""
			}
		}
	}
}

// isUpgradePlan returns true if this could be an upgrade plan - this is just an approximation because deploy plan can be used for both
func isUpgradePlan(planName string) bool {
	return planName == v1beta1.DeployPlanName || planName == v1beta1.UpgradePlanName
}

// getParamDefinitions retrieves param definitions of the passed in map keys from the OperatorVersion
func getParamDefinitions(params map[string]string, ov *v1beta1.OperatorVersion) []v1beta1.Parameter {
	defs := []v1beta1.Parameter{}
	for p1 := range params {
		for _, p2 := range ov.Spec.Parameters {
			if p2.Name == p1 {
				defs = append(defs, p2)
			}
		}
	}
	return defs
}

// parameterDifference returns map containing all parameters that were removed or changed between old and new
func parameterDifference(old, new map[string]string) map[string]string {
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

// planNameFromParameters determines what plan to run based on params that changed and the related trigger plans
func planNameFromParameters(params []v1beta1.Parameter, ov *v1beta1.OperatorVersion) *string {
	for _, p := range params {
		// TODO: if the params have different trigger plans, we always select first here which might not be ideal
		if p.Trigger != "" && selectPlan([]string{p.Trigger}, ov) != nil {
			return kudo.String(p.Trigger)
		}
	}
	return selectPlan([]string{v1beta1.UpdatePlanName, v1beta1.DeployPlanName}, ov)
}

// selectPlan returns nil if none of the plan exists, otherwise the first one in list that exists
func selectPlan(possiblePlans []string, ov *v1beta1.OperatorVersion) *string {
	for _, n := range possiblePlans {
		if _, ok := ov.Spec.Plans[n]; ok {
			return kudo.String(n)
		}
	}
	return nil
}

func (op reconcileOperation) PlanStatus(plan string) *v1beta1.PlanStatus {
	for _, planStatus := range op.instance.Status.PlanStatus {
		if planStatus.Name == plan {
			return &planStatus
		}
	}
	return nil
}

// GetPlanInProgress returns plan status of currently active plan or nil if no plan is running
func (op reconcileOperation) GetPlanInProgress() *v1beta1.PlanStatus {
	for _, p := range op.instance.Status.PlanStatus {
		if p.Status.IsRunning() {
			return &p
		}
	}
	return nil
}

// NoPlanEverExecuted returns true is this is new instance for which we never executed any plan
func (op reconcileOperation) NoPlanEverExecuted() bool {
	for _, p := range op.instance.Status.PlanStatus {
		if p.Status != v1beta1.ExecutionNeverRun {
			return false
		}
	}
	return true
}

// TryAddFinalizerToInstance adds the cleanup finalizer to an instance if the finalizer
// hasn't been added yet, the instance has a cleanup plan and the cleanup plan
// didn't run yet. Returns true if the cleanup finalizer has been added.
func (op reconcileOperation) TryAddFinalizerToInstance() bool {
	if !contains(op.instance.ObjectMeta.Finalizers, v1beta1.InstanceCleanupFinalizerName) {
		if planStatus := op.PlanStatus(v1beta1.CleanupPlanName); planStatus != nil {
			// avoid adding a finalizer again if a reconciliation is requested
			// after it has just been removed but the instance isn't deleted yet
			if planStatus.Status == v1beta1.ExecutionNeverRun {
				op.instance.ObjectMeta.Finalizers = append(op.instance.ObjectMeta.Finalizers, v1beta1.InstanceCleanupFinalizerName)
				return true
			}
		}
	}

	return false
}

// TryRemoveFinalizer removes the cleanup finalizer of an instance if it has
// been added, the instance has a cleanup plan and the cleanup plan completed.
// Returns true if the cleanup finalizer has been removed.
func (op reconcileOperation) TryRemoveFinalizer() bool {
	if contains(op.instance.ObjectMeta.Finalizers, v1beta1.InstanceCleanupFinalizerName) {
		if planStatus := op.PlanStatus(v1beta1.CleanupPlanName); planStatus != nil {
			if planStatus.Status.IsTerminal() {
				op.instance.ObjectMeta.Finalizers = remove(op.instance.ObjectMeta.Finalizers, v1beta1.InstanceCleanupFinalizerName)
				return true
			}
		} else {
			// We have a finalizer but no cleanup plan. This could be due to an updated instance.
			// Let's remove the finalizer.
			op.instance.ObjectMeta.Finalizers = remove(op.instance.ObjectMeta.Finalizers, v1beta1.InstanceCleanupFinalizerName)
			return true
		}
	}

	return false
}

func remove(values []string, s string) (result []string) {
	for _, value := range values {
		if value == s {
			continue
		}
		result = append(result, value)
	}
	return
}

// SaveSnapshot stores the current spec of Instance into the snapshot annotation
// this information is used when executing update/upgrade plans, this overrides any snapshot that existed before
func (op reconcileOperation) SaveSnapshot() error {
	jsonBytes, err := json.Marshal(op.instance.Spec)
	if err != nil {
		return err
	}
	if op.instance.Annotations == nil {
		op.instance.Annotations = make(map[string]string)
	}
	op.instance.Annotations[snapshotAnnotation] = string(jsonBytes)
	return nil
}

func (op reconcileOperation) snapshotSpec() (*v1beta1.InstanceSpec, error) {
	if op.instance.Annotations != nil {
		snapshot, ok := op.instance.Annotations[snapshotAnnotation]
		if ok {
			var spec *v1beta1.InstanceSpec
			err := json.Unmarshal([]byte(snapshot), &spec)
			if err != nil {
				return nil, err
			}
			return spec, nil
		}
	}
	return nil, nil
}

func contains(values []string, s string) bool {
	for _, value := range values {
		if value == s {
			return true
		}
	}
	return false
}

// getInstance retrieves the instance by namespaced name
// returns nil, nil when instance is not found (not found is not considered an error)
func (op reconcileOperation) getInstance(request ctrl.Request) (instance *v1beta1.Instance, err error) {
	instance = &kudov1beta1.Instance{}
	err = op.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		// Error reading the object - requeue the request.
		log.Printf("InstanceController: Error getting instance \"%v\": %v",
			request.NamespacedName,
			err)
		return nil, err
	}
	return instance, nil
}

// getOperatorVersion retrieves operatorversion belonging to the given instance
// not found is treated here as any other error
func (op reconcileOperation) getOperatorVersion(instance *kudov1beta1.Instance) (ov *kudov1beta1.OperatorVersion, err error) {
	ov = &kudov1beta1.OperatorVersion{}
	err = op.client.Get(context.TODO(),
		types.NamespacedName{
			Name:      instance.Spec.OperatorVersion.Name,
			Namespace: instance.OperatorVersionNamespace(),
		},
		ov)
	if err != nil {
		log.Printf("InstanceController: Error getting operatorVersion \"%v\" for instance \"%v\": %v",
			instance.Spec.OperatorVersion.Name,
			instance.Name,
			err)
		op.recorder.Event(instance, "Warning", "InvalidOperatorVersion", fmt.Sprintf("Error getting operatorVersion \"%v\": %v", instance.Spec.OperatorVersion.Name, err))
		return nil, err
	}
	return ov, nil
}
