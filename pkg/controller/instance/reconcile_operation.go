package instance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	kudov1beta1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/apitools/instance"
	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

const (
	instanceCleanupFinalizerName = "kudo.dev.instance.cleanup"
	snapshotAnnotation           = "kudo.dev/last-applied-instance-state"
)

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

func (op reconcileOperation) loadInstance(req ctrl.Request) error {
	inst, err := fetchInstance(op.client, req)
	if err != nil {
		return err
	}
	op.instance = inst
	op.oldInstance = inst.DeepCopy()
	return nil
}

func (op reconcileOperation) loadOperatorVersion() error {
	ov, err := fetchOperatorVersion(op.client, op.recorder, op.instance)
	if err != nil {
		return err
	}

	op.ov = ov
	return nil
}

// GetPlanToBeExecuted returns name of the plan that should be executed
func (op reconcileOperation) GetPlanToBeExecuted() (*string, error) {
	if instance.IsDeleting(op.instance) {
		// we have a cleanup plan
		plan := selectPlan([]string{v1beta1.CleanupPlanName}, op.ov)
		if plan != nil {
			if planStatus := instance.PlanStatus(op.instance, *plan); planStatus != nil {
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

	if instance.GetPlanInProgress(op.instance) != nil { // we're already running some plan
		return nil, nil
	}

	// new instance, need to run deploy plan
	if instance.NoPlanEverExecuted(op.instance) {
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
	if instance.NoPlanEverExecuted(op.instance) || isUpgradePlan(planName) {
		op.ensurePlanStatusInitialized()
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

	err := op.saveSnapshot()
	if err != nil {
		return err
	}

	return nil
}

// updateInstance saves the modified instance to K8s if it has changed
func (op reconcileOperation) updateInstance() error {

	// update instance spec and metadata. this will not update Instance.Status field
	if !reflect.DeepEqual(op.instance.Spec, op.oldInstance.Spec) ||
		!reflect.DeepEqual(op.instance.ObjectMeta.Annotations, op.oldInstance.ObjectMeta.Annotations) ||
		!reflect.DeepEqual(op.instance.ObjectMeta.Finalizers, op.oldInstance.ObjectMeta.Finalizers) {
		instanceStatus := op.instance.Status.DeepCopy()
		err := op.client.Update(context.TODO(), op.instance)
		if err != nil {
			log.Printf("InstanceController: Error when updating instance spec. %v", err)
			return err
		}
		op.instance.Status = *instanceStatus
	}

	// update instance status
	err := op.client.Status().Update(context.TODO(), op.instance)
	if err != nil {
		log.Printf("InstanceController: Error when updating instance status. %v", err)
		return err
	}

	// update instance metadata if finalizer is removed
	// because Kubernetes might immediately delete the instance, this has to be the last instance update
	if op.TryRemoveFinalizer() {
		log.Printf("InstanceController: Removing finalizer on instance %s/%s", op.instance.Namespace, op.instance.Name)
		if err := op.client.Update(context.TODO(), op.instance); err != nil {
			log.Printf("InstanceController: Error when removing instance finalizer. %v", err)
			return err
		}
	}

	return nil
}

// handleError handles execution error by logging, updating the plan status and optionally publishing an event
// specify eventReason as nil if you don't wish to publish a warning event
// returns err if this err should be retried, nil otherwise
func (op reconcileOperation) handleError(err error) error {
	inst := op.instance
	log.Printf("InstanceController: %v", err)

	// first update instance as we want to propagate errors also to the `Instance.Status.PlanStatus`
	clientErr := op.updateInstance()
	if clientErr != nil {
		log.Printf("InstanceController: Error when updating instance state. %v", clientErr)
		return clientErr
	}

	// determine if retry is necessary based on the error type
	var exErr engine.ExecutionError
	if errors.As(err, &exErr) {
		op.recorder.Event(inst, "Warning", exErr.EventName, err.Error())

		if errors.Is(exErr, engine.ErrFatalExecution) {
			return nil // not retrying fatal error
		}
	}

	// for code being processed on instance, we need to handle these errors as well
	var iError *kudov1beta1.InstanceError
	if errors.As(err, &iError) {
		if iError.EventName != nil {
			op.recorder.Event(inst, "Warning", kudo.StringValue(iError.EventName), err.Error())
		}
	}
	return err
}

// ensurePlanStatusInitialized initializes plan status for all plans this instance supports
// it does not trigger run of any plan
// it either initializes everything for a fresh instance without any status or tries to adjust status after OV was updated
func (op reconcileOperation) ensurePlanStatusInitialized() {
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

// TryAddFinalizerToInstance adds the cleanup finalizer to an instance if the finalizer
// hasn't been added yet, the instance has a cleanup plan and the cleanup plan
// didn't run yet. Returns true if the cleanup finalizer has been added.
func (op reconcileOperation) TryAddFinalizerToInstance() bool {
	if !contains(op.instance.ObjectMeta.Finalizers, instanceCleanupFinalizerName) {
		if planStatus := instance.PlanStatus(op.instance, v1beta1.CleanupPlanName); planStatus != nil {
			// avoid adding a finalizer again if a reconciliation is requested
			// after it has just been removed but the instance isn't deleted yet
			if planStatus.Status == v1beta1.ExecutionNeverRun {
				op.instance.ObjectMeta.Finalizers = append(op.instance.ObjectMeta.Finalizers, instanceCleanupFinalizerName)
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
	if contains(op.instance.ObjectMeta.Finalizers, instanceCleanupFinalizerName) {
		if planStatus := instance.PlanStatus(op.instance, v1beta1.CleanupPlanName); planStatus != nil {
			if planStatus.Status.IsTerminal() {
				op.instance.ObjectMeta.Finalizers = remove(op.instance.ObjectMeta.Finalizers, instanceCleanupFinalizerName)
				return true
			}
		} else {
			// We have a finalizer but no cleanup plan. This could be due to an updated instance.
			// Let's remove the finalizer.
			op.instance.ObjectMeta.Finalizers = remove(op.instance.ObjectMeta.Finalizers, instanceCleanupFinalizerName)
			return true
		}
	}

	return false
}

func contains(values []string, s string) bool {
	for _, value := range values {
		if value == s {
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

// saveSnapshot stores the current spec of Instance into the snapshot annotation
// this information is used when executing update/upgrade plans, this overrides any snapshot that existed before
func (op reconcileOperation) saveSnapshot() error {
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

// snapshotSpec returns a saved snapshot as an InstanceSpec
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

// fetchInstance retrieves the instance by namespaced name
// returns nil, nil when instance is not found (not found is not considered an error)
func fetchInstance(client client.Client, request ctrl.Request) (instance *v1beta1.Instance, err error) {
	instance = &kudov1beta1.Instance{}
	err = client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		// Error reading the object - requeue the request.
		log.Printf("InstanceController: Error getting instance \"%v\": %v",
			request.NamespacedName,
			err)
		return nil, err
	}
	return instance, nil
}

// fetchOperatorVersion retrieves operatorversion belonging to the given instance
// not found is treated here as any other error
func fetchOperatorVersion(client client.Client, recorder record.EventRecorder, in *kudov1beta1.Instance) (ov *kudov1beta1.OperatorVersion, err error) {
	ov = &kudov1beta1.OperatorVersion{}
	err = client.Get(context.TODO(),
		types.NamespacedName{
			Name:      in.Spec.OperatorVersion.Name,
			Namespace: instance.OperatorVersionNamespace(in),
		},
		ov)
	if err != nil {
		log.Printf("InstanceController: Error getting operatorVersion \"%v\" for instance \"%v\": %v",
			in.Spec.OperatorVersion.Name,
			in.Name,
			err)
		recorder.Event(in, "Warning", "InvalidOperatorVersion", fmt.Sprintf("Error getting operatorVersion \"%v\": %v", in.Spec.OperatorVersion.Name, err))
		return nil, err
	}
	return ov, nil
}
