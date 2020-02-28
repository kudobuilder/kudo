/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package instance

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"

	"github.com/thoas/go-funk"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	kudov1beta1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/engine/renderer"
	"github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/engine/workflow"
	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

// Reconciler reconciles an Instance object.
type Reconciler struct {
	client.Client
	Discovery discovery.DiscoveryInterface
	Config    *rest.Config
	Recorder  record.EventRecorder
	Scheme    *runtime.Scheme
}

// SetupWithManager registers this reconciler with the controller manager
func (r *Reconciler) SetupWithManager(
	mgr ctrl.Manager) error {
	addOvRelatedInstancesToReconcile := handler.ToRequestsFunc(
		func(obj handler.MapObject) []reconcile.Request {
			requests := make([]reconcile.Request, 0)
			instances := &kudov1beta1.InstanceList{}
			// we are listing all instances here, which could come with some performance penalty
			// obj possible optimization is to introduce filtering based on operatorversion (or operator)
			err := mgr.GetClient().List(
				context.TODO(),
				instances,
			)
			if err != nil {
				log.Printf("InstanceController: Error fetching instances list for operator %v: %v", obj.Meta.GetName(), err)
				return nil
			}
			for _, instance := range instances.Items {
				// we need to pick only those instances, that belong to the OperatorVersion we're reconciling
				if instance.Spec.OperatorVersion.Name == obj.Meta.GetName() &&
					instance.OperatorVersionNamespace() == obj.Meta.GetNamespace() {
					requests = append(requests, reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      instance.Name,
							Namespace: instance.Namespace,
						},
					})
				}
			}
			return requests
		})

	return ctrl.NewControllerManagedBy(mgr).
		For(&kudov1beta1.Instance{}).
		Owns(&kudov1beta1.Instance{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&batchv1.Job{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Pod{}).
		WithEventFilter(eventFilter()).
		Watches(&source.Kind{Type: &kudov1beta1.OperatorVersion{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: addOvRelatedInstancesToReconcile}).
		Complete(r)
}

// eventFilter ignores DeleteEvents for pipe-pods only (marked with task.PipePodAnnotation). This is due to an
// inherent race that was described in detail in #1116 (https://github.com/kudobuilder/kudo/issues/1116)
// tl;dr: pipe-task will delete the pipe pod at the end of the execution. this would normally trigger another
// Instance reconciliation which might end up copying pipe files twice. we avoid this by explicitly ignoring
// DeleteEvents for pipe-pods.
func eventFilter() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(event.CreateEvent) bool { return true },
		DeleteFunc: func(e event.DeleteEvent) bool {
			return !isForPipePod(e)
		},
		UpdateFunc:  func(event.UpdateEvent) bool { return true },
		GenericFunc: func(event.GenericEvent) bool { return true },
	}
}

func isForPipePod(e event.DeleteEvent) bool {
	return e.Meta.GetAnnotations() != nil && funk.Contains(e.Meta.GetAnnotations(), task.PipePodAnnotation)
}

// Reconcile is the main controller method that gets called every time something about the instance changes
//
//   +-------------------------------+
//   | Query state of Instance       |
//   | and OperatorVersion           |
//   +-------------------------------+
//                  |
//                  v
//   +-------------------------------+
//   | Update finalizers if cleanup  |
//   | plan exists                   |
//   +-------------------------------+
//                  |
//                  v
//   +-------------------------------+
//   | Start new plan if required    |
//   | and none is running           |
//   +-------------------------------+
//                  |
//                  v
//   +-------------------------------+
//   | If there is plan in progress, |
//   | proceed with the execution    |
//   +-------------------------------+
//                  |
//                  v
//   +-------------------------------+
//   | Update instance with new      |
//   | state of the execution        |
//   +-------------------------------+
//
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
func (r *Reconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	// ---------- 1. Query the current state ----------

	log.Printf("InstanceController: Received Reconcile request for instance \"%+v\"", request.Name)
	instance, err := r.getInstance(request)
	if err != nil {
		if apierrors.IsNotFound(err) { // not retrying if instance not found, probably someone manually removed it?
			log.Printf("Instance %s was deleted, nothing to reconcile.", request.NamespacedName)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	oldInstance := instance.DeepCopy()

	ov, err := GetOperatorVersion(instance, r.Client)
	if err != nil {
		err = fmt.Errorf("InstanceController: Error getting operatorVersion %s for instance %s/%s: %v",
			instance.Spec.OperatorVersion.Name, instance.Namespace, instance.Name, err)
		log.Print(err)
		r.Recorder.Event(instance, "Warning", "InvalidOperatorVersion", err.Error())
		return reconcile.Result{}, err // OV not found has to be retried because it can really have been created after Instance
	}

	// ---------- 2. Try to add a finalizer (effectively happens only once after creation) ----------

	if !instance.IsDeleting() {
		if _, hasCleanupPlan := ov.Spec.Plans[kudov1beta1.CleanupPlanName]; hasCleanupPlan {
			if instance.TryAddFinalizer() {
				log.Printf("InstanceController: Adding finalizer on instance %s/%s", instance.Namespace, instance.Name)
			}
		}
	}

	// ---------- 3. Check if we should start execution of new plan ----------

	newExecutionPlan, err := newExecutionPlan(instance, ov)
	if err != nil {
		return reconcile.Result{}, err
	}

	if newExecutionPlan != nil {
		log.Printf("InstanceController: Going to start execution of plan %s on instance %s/%s", kudo.StringValue(newExecutionPlan), instance.Namespace, instance.Name)
		err = startPlanExecution(instance, kudo.StringValue(newExecutionPlan), ov)
		if err != nil {
			return reconcile.Result{}, r.handleError(err, instance, oldInstance)
		}
		r.Recorder.Event(instance, "Normal", "PlanStarted", fmt.Sprintf("Execution of plan %s started", kudo.StringValue(newExecutionPlan)))
	}

	// ---------- 4. If there's currently active plan, continue with the execution ----------

	activePlanStatus := getPlanInProgress(instance)
	if activePlanStatus == nil { // we have no plan in progress
		log.Printf("InstanceController: Nothing to do, no plan in progress for instance %s/%s", instance.Namespace, instance.Name)
		return reconcile.Result{}, nil
	}

	metadata := &engine.Metadata{
		OperatorVersionName: ov.Name,
		OperatorVersion:     ov.Spec.Version,
		AppVersion:          ov.Spec.AppVersion,
		ResourcesOwner:      instance,
		OperatorName:        ov.Spec.Operator.Name,
		InstanceNamespace:   instance.Namespace,
		InstanceName:        instance.Name,
	}

	activePlan, err := preparePlanExecution(instance, ov, activePlanStatus, metadata)
	if err != nil {
		err = r.handleError(err, instance, oldInstance)
		return reconcile.Result{}, err
	}
	log.Printf("InstanceController: Going to proceed in execution of active plan %s on instance %s/%s", activePlan.Name, instance.Namespace, instance.Name)
	newStatus, err := workflow.Execute(activePlan, metadata, r.Client, r.Discovery, r.Config, &renderer.DefaultEnhancer{Scheme: r.Scheme, Discovery: r.Discovery}, time.Now())

	// ---------- 5. Update instance and its status after the execution proceeded ----------

	if newStatus != nil {
		instance.UpdateInstanceStatus(newStatus)
	}
	if err != nil {
		err = r.handleError(err, instance, oldInstance)
		return reconcile.Result{}, err
	}

	err = updateInstance(instance, oldInstance, r.Client)
	if err != nil {
		log.Printf("InstanceController: Error when updating instance. %v", err)
		return reconcile.Result{}, err
	}

	// Publish a PlanFinished event after instance and its status were successfully updated
	if instance.Status.AggregatedStatus.Status.IsTerminal() {
		r.Recorder.Event(instance, "Normal", "PlanFinished", fmt.Sprintf("Execution of plan %s finished with status %s", activePlanStatus.Name, instance.Status.AggregatedStatus.Status))
	}

	return reconcile.Result{}, nil
}

func updateInstance(instance *kudov1beta1.Instance, oldInstance *kudov1beta1.Instance, client client.Client) error {
	// update instance spec and metadata. this will not update Instance.Status field
	if !reflect.DeepEqual(instance.Spec, oldInstance.Spec) ||
		!reflect.DeepEqual(instance.ObjectMeta.Annotations, oldInstance.ObjectMeta.Annotations) ||
		!reflect.DeepEqual(instance.ObjectMeta.Finalizers, oldInstance.ObjectMeta.Finalizers) {
		instanceStatus := instance.Status.DeepCopy()
		err := client.Update(context.TODO(), instance)
		if err != nil {
			log.Printf("InstanceController: Error when updating instance spec. %v", err)
			return err
		}
		instance.Status = *instanceStatus
	}

	// update instance status
	err := client.Status().Update(context.TODO(), instance)
	if err != nil {
		log.Printf("InstanceController: Error when updating instance status. %v", err)
		return err
	}

	// update instance metadata if finalizer is removed
	// because Kubernetes might immediately delete the instance, this has to be the last instance update
	if instance.TryRemoveFinalizer() {
		log.Printf("InstanceController: Removing finalizer on instance %s/%s", instance.Namespace, instance.Name)
		if err := client.Update(context.TODO(), instance); err != nil {
			log.Printf("InstanceController: Error when removing instance finalizer. %v", err)
			return err
		}
	}

	return nil
}

func preparePlanExecution(instance *kudov1beta1.Instance, ov *kudov1beta1.OperatorVersion, activePlanStatus *kudov1beta1.PlanStatus, meta *engine.Metadata) (*workflow.ActivePlan, error) {
	planSpec, ok := ov.Spec.Plans[activePlanStatus.Name]
	if !ok {
		return nil, &engine.ExecutionError{Err: fmt.Errorf("%wcould not find required plan: %v", engine.ErrFatalExecution, activePlanStatus.Name), EventName: "InvalidPlan"}
	}

	params := paramsMap(instance, ov)
	pipes, err := pipesMap(activePlanStatus.Name, &planSpec, ov.Spec.Tasks, meta)
	if err != nil {
		return nil, &engine.ExecutionError{Err: fmt.Errorf("%wcould not make task pipes: %v", engine.ErrFatalExecution, err), EventName: "InvalidPlan"}
	}

	return &workflow.ActivePlan{
		Name:       activePlanStatus.Name,
		Spec:       &planSpec,
		PlanStatus: activePlanStatus,
		Tasks:      ov.Spec.Tasks,
		Templates:  ov.Spec.Templates,
		Params:     params,
		Pipes:      pipes,
	}, nil
}

// handleError handles execution error by logging, updating the plan status and optionally publishing an event
// specify eventReason as nil if you don't wish to publish a warning event
// returns err if this err should be retried, nil otherwise
func (r *Reconciler) handleError(err error, instance *kudov1beta1.Instance, oldInstance *kudov1beta1.Instance) error {
	log.Printf("InstanceController: %v", err)

	// first update instance as we want to propagate errors also to the `Instance.Status.PlanStatus`
	clientErr := updateInstance(instance, oldInstance, r.Client)
	if clientErr != nil {
		log.Printf("InstanceController: Error when updating instance state. %v", clientErr)
		return clientErr
	}

	// determine if retry is necessary based on the error type
	var exErr engine.ExecutionError
	if errors.As(err, &exErr) {
		r.Recorder.Event(instance, "Warning", exErr.EventName, err.Error())

		if errors.Is(exErr, engine.ErrFatalExecution) {
			return nil // not retrying fatal error
		}
	}

	// for code being processed on instance, we need to handle these errors as well
	var iError *kudov1beta1.InstanceError
	if errors.As(err, &iError) {
		if iError.EventName != nil {
			r.Recorder.Event(instance, "Warning", kudo.StringValue(iError.EventName), err.Error())
		}
	}
	return err
}

// getInstance retrieves the instance by namespaced name
func (r *Reconciler) getInstance(request ctrl.Request) (instance *kudov1beta1.Instance, err error) {
	instance = &kudov1beta1.Instance{}
	err = r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		// Error reading the object - requeue the request.
		log.Printf("InstanceController: Error getting instance \"%v\": %v",
			request.NamespacedName,
			err)
		return nil, err
	}
	return instance, nil
}

// GetOperatorVersion retrieves OperatorVersion belonging to the given instance
func GetOperatorVersion(instance *kudov1beta1.Instance, c client.Reader) (ov *kudov1beta1.OperatorVersion, err error) {
	ov = &kudov1beta1.OperatorVersion{}
	err = c.Get(context.TODO(),
		types.NamespacedName{
			Name:      instance.Spec.OperatorVersion.Name,
			Namespace: instance.OperatorVersionNamespace(),
		},
		ov)
	if err != nil {
		return nil, err
	}
	return ov, nil
}

// paramsMap generates {{ Params.* }} map of keys and values which is later used during template rendering.
func paramsMap(instance *kudov1beta1.Instance, operatorVersion *kudov1beta1.OperatorVersion) map[string]string {
	params := make(map[string]string)

	for k, v := range instance.Spec.Parameters {
		params[k] = v
	}

	// Merge instance parameter overrides with operator version, if no override exist, use the default one
	for _, param := range operatorVersion.Spec.Parameters {
		if _, ok := params[param.Name]; !ok {
			params[param.Name] = kudo.StringValue(param.Default)
		}
	}

	return params
}

// pipesMap generates {{ Pipes.* }} map of keys and values which is later used during template rendering.
func pipesMap(planName string, plan *v1beta1.Plan, tasks []v1beta1.Task, emeta *engine.Metadata) (map[string]string, error) {
	taskByName := func(name string) (*v1beta1.Task, bool) {
		for _, t := range tasks {
			if t.Name == name {
				return &t, true
			}
		}
		return nil, false
	}

	pipes := make(map[string]string)

	for _, ph := range plan.Phases {
		for _, st := range ph.Steps {
			for _, tn := range st.Tasks {
				rmeta := renderer.Metadata{
					Metadata:  *emeta,
					PlanName:  planName,
					PhaseName: ph.Name,
					StepName:  st.Name,
					TaskName:  tn,
				}

				if t, ok := taskByName(tn); ok && t.Kind == task.PipeTaskKind {
					for _, pipe := range t.Spec.PipeTaskSpec.Pipe {
						if _, ok := pipes[pipe.Key]; ok {
							return nil, fmt.Errorf("duplicated pipe key %s", pipe.Key)
						}
						pipes[pipe.Key] = task.PipeArtifactName(rmeta, pipe.Key)
					}
				}
			}
		}
	}

	return pipes, nil
}

// startPlanExecution mark plan as to be executed
// this modifies the instance.Status as well as instance.Metadata.Annotation (to save snapshot if needed)
func startPlanExecution(i *v1beta1.Instance, planName string, ov *v1beta1.OperatorVersion) error {
	if i.NoPlanEverExecuted() || isUpgradePlan(planName) {
		ensurePlanStatusInitialized(i, ov)
	}

	// reset newly starting plan status
	if err := i.ResetPlanStatus(planName); err != nil {
		return &v1beta1.InstanceError{Err: fmt.Errorf("failed to reset plan status for instance %s/%s: %v", i.Namespace, i.Name, err), EventName: kudo.String("PlanNotFound")}
	}

	err := i.AnnotateSnapshot()
	if err != nil {
		return err
	}

	return nil
}

// ensurePlanStatusInitialized initializes plan status for all plans this instance supports
// it does not trigger run of any plan
// it either initializes everything for a fresh instance without any status or tries to adjust status after OV was updated
func ensurePlanStatusInitialized(i *v1beta1.Instance, ov *v1beta1.OperatorVersion) {
	if i.Status.PlanStatus == nil {
		i.Status.PlanStatus = make(map[string]v1beta1.PlanStatus)
	}

	for planName, plan := range ov.Spec.Plans {
		planStatus := &v1beta1.PlanStatus{
			Name:   planName,
			Status: v1beta1.ExecutionNeverRun,
			Phases: make([]v1beta1.PhaseStatus, 0),
		}

		existingPlanStatus, planExists := i.Status.PlanStatus[planName]
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
		i.Status.PlanStatus[planName] = *planStatus
	}
}

// isUpgradePlan returns true if this could be an upgrade plan - this is just an approximation because deploy plan can be used for both
func isUpgradePlan(planName string) bool {
	return planName == v1beta1.DeployPlanName || planName == v1beta1.UpgradePlanName
}

func areWebhooksEnabled() bool {
	return strings.ToLower(os.Getenv("ENABLE_WEBHOOKS")) == "true"
}

// getPlanInProgress method returns current plan that is in progress. As long as we don't enforce webhooks
// (ENABLE_WEBHOOKS=true) we have TWO WAYS of deciding that:
//
// 1. WITHOUT the webhook, we have the old logic which iterates over all existing PlanStatuses, searching for
//    the first one with Status == InProgress. It might happen that multiple plans are in progress (e.g. cleanup and deploy)
// 2. WITH the webhook, current plan lives in the Spec.PlanExecution.PlanName field
func getPlanInProgress(i *v1beta1.Instance) *kudov1beta1.PlanStatus {
	if areWebhooksEnabled() {
		return i.PlanStatus(i.Spec.PlanExecution.PlanName)
	}
	return i.GetPlanInProgress()
}

// newExecutionPlan method returns a new execution plan (if exists) or nil otherwise. As long as we don't enforce webhooks
// (ENABLE_WEBHOOKS=true) we have TWO WAYS of deciding which plan has to executed next:
//
// 1. WITHOUT the webhook, we have the old logic which tries to infer the change (parameter update, new OperatorVersion etc.)
//    by diffing current state with the one from the snapshot.
// 2. WITH the webhook, instance admission webhook has already decided on the plan and the result is in the
//    Spec.PlanExecution.PlanName field.
func newExecutionPlan(i *v1beta1.Instance, ov *v1beta1.OperatorVersion) (plan *string, err error) {
	if areWebhooksEnabled() {
		if plan, err = fetchNewExecutionPlan(i, ov); plan != nil && err == nil {
			log.Printf("InstanceController: Fetched new execution plan '%s' from the spec for instance %s/%s", kudo.StringValue(plan), i.Namespace, i.Name)
		}
	} else {
		if plan, err = inferNewExecutionPlan(i, ov); plan != nil && err == nil {
			log.Printf("InstanceController: Inferred new execution plan '%s' from instance %s/%s state", kudo.StringValue(plan), i.Namespace, i.Name)
		}
	}

	return
}

// fetchNewExecutionPlan method fetches a new execution plan from Instance.Spec.PlanExecution field. A plan is actually
// new if the PlanExecution.UID has changed, otherwise it's just an old plan in progress.
// - "planName", when there is a new plan that needs to be executed
// - <nil>, no new plan found e.g. a plan is already in progress
func fetchNewExecutionPlan(i *v1beta1.Instance, ov *v1beta1.OperatorVersion) (*string, error) {
	snapshot, err := i.SnapshotSpec()
	if err != nil {
		return nil, &v1beta1.InstanceError{Err: fmt.Errorf("failed to unmarshal instance snapshot %s/%s: %v", i.Namespace, i.Name, err), EventName: kudo.String("UnexpectedState")}
	}

	// Instance deletion is an edge case where the admission webhook *can not* populate the Spec.PlanExecution.PlanName
	// with the 'cleanup' plan. So we have to do it here ourselves. Only if:
	// 1. Instance is being deleted
	// 2. Cleanup plan exists in the operator version and has *never run* before
	// 3. Cleanup hasn't been scheduled yet (first time the deletion is being reconciled)
	// we set the Spec.PlanExecution.PlanName = 'cleanup'
	hasToScheduleCleanupAfterDeletion := func() bool {
		shouldCleanup := i.IsDeleting() && kudov1beta1.CleanupPlanExists(ov)
		cleanupNeverRun := i.PlanStatus(v1beta1.CleanupPlanName) == nil || i.PlanStatus(v1beta1.CleanupPlanName).Status == kudov1beta1.ExecutionNeverRun
		cleanupNotScheduled := i.Spec.PlanExecution.PlanName != v1beta1.CleanupPlanName

		return shouldCleanup && cleanupNeverRun && cleanupNotScheduled
	}
	if hasToScheduleCleanupAfterDeletion() {
		log.Printf("InstanceController: Instance %s/%s is being deleted. Scheduling '%s' plan.", i.Namespace, i.Name, v1beta1.CleanupPlanName)

		i.Spec.PlanExecution.PlanName = v1beta1.CleanupPlanName
		i.Spec.PlanExecution.UID = uuid.NewUUID()
	}

	newPlanScheduled := func() bool {
		oldUID := snapshot.PlanExecution.UID
		newUID := i.Spec.PlanExecution.UID
		isNovelPlan := oldUID == "" && newUID != ""
		isPlanOverride := oldUID != "" && newUID != "" && newUID != oldUID

		return isNovelPlan || isPlanOverride
	}

	if newPlanScheduled() {
		return &i.Spec.PlanExecution.PlanName, nil
	}

	return nil, nil
}

// newPlanToBeExecuted method tries to infer a new execution plan by comparing current instance state
// with the one saved in the snapshot. It returns:
// - "planName", when there is a new plan that needs to be executed
// - <nil>, no new plan found e.g. a plan is already in progress
func inferNewExecutionPlan(i *v1beta1.Instance, ov *v1beta1.OperatorVersion) (*string, error) {
	if i.IsDeleting() {
		log.Printf("InstanceController: Instance %s/%s is being deleted", i.Namespace, i.Name)
		// we have a cleanup plan
		cleanupPlanName := v1beta1.CleanupPlanName
		if kudov1beta1.PlanExists(cleanupPlanName, ov) {
			if planStatus := i.PlanStatus(cleanupPlanName); planStatus != nil {
				switch planStatus.Status {
				case kudov1beta1.ExecutionNeverRun:
					return &cleanupPlanName, nil
				case kudov1beta1.ExecutionComplete, kudov1beta1.ExecutionFatalError:
					return nil, nil // we already finished the cleanup plan or there is no point in retrying
				}
			}
		}
	}

	if i.GetPlanInProgress() != nil { // we're already running some plan
		return nil, nil
	}

	// new instance, need to run deploy plan
	if i.NoPlanEverExecuted() {
		return kudo.String(v1beta1.DeployPlanName), nil
	}

	// did the instance change so that we need to run deploy/upgrade/update plan?
	instanceSnapshot, err := i.SnapshotSpec()
	if err != nil {
		return nil, err
	}
	if instanceSnapshot == nil {
		// we don't have snapshot -> we never run deploy, also we cannot run update/upgrade. This should never happen
		return nil, &v1beta1.InstanceError{Err: fmt.Errorf("unexpected state: no plan is running, no snapshot present - this should never happen :) for instance %s/%s", i.Namespace, i.Name), EventName: kudo.String("UnexpectedState")}
	}
	if instanceSnapshot.OperatorVersion.Name != i.Spec.OperatorVersion.Name {
		// this instance was upgraded to newer version
		log.Printf("Instance: instance %s/%s was upgraded from %s to %s operatorVersion", i.Namespace, i.Name, instanceSnapshot.OperatorVersion.Name, i.Spec.OperatorVersion.Name)
		plan := kudov1beta1.SelectPlan([]string{v1beta1.UpgradePlanName, v1beta1.UpdatePlanName, v1beta1.DeployPlanName}, ov)
		if plan == nil {
			return nil, &v1beta1.InstanceError{Err: fmt.Errorf("supposed to execute plan because instance %s/%s was upgraded but none of the deploy, upgrade, update plans found in linked operatorVersion", i.Namespace, i.Name),
				EventName: kudo.String("PlanNotFound")}
		}
		return plan, nil
	}
	// did instance parameters change, so that the corresponding plan has to be triggered?
	if !reflect.DeepEqual(instanceSnapshot.Parameters, i.Spec.Parameters) {
		// instance updated
		log.Printf("Instance: instance %s/%s has updated parameters from %v to %v", i.Namespace, i.Name, instanceSnapshot.Parameters, i.Spec.Parameters)
		paramDiff := kudov1beta1.ParameterDiff(instanceSnapshot.Parameters, i.Spec.Parameters)
		paramDefinitions := kudov1beta1.GetParamDefinitions(paramDiff, ov)
		plan, err := planNameFromParameters(paramDefinitions, ov)
		if err != nil {
			return nil, &v1beta1.InstanceError{Err: fmt.Errorf("supposed to execute plan because instance %s/%s was updated but no valid plan found: %v", i.Namespace, i.Name, err), EventName: kudo.String("PlanNotFound")}
		}
		return plan, nil
	}
	return nil, nil
}

// planNameFromParameters determines what plan to run based on params that changed and the related trigger plans
func planNameFromParameters(params []v1beta1.Parameter, ov *v1beta1.OperatorVersion) (*string, error) {
	// TODO: if the params have different trigger plans, we always select first here which might not be ideal
	for _, p := range params {
		if p.Trigger != "" {
			if kudov1beta1.SelectPlan([]string{p.Trigger}, ov) != nil {
				return kudo.String(p.Trigger), nil
			}
			return nil, fmt.Errorf("param %s defined trigger plan %s, but plan not defined in operatorversion", p.Name, p.Trigger)
		}
	}
	plan := kudov1beta1.SelectPlan([]string{v1beta1.UpdatePlanName, v1beta1.DeployPlanName}, ov)
	if plan == nil {
		return nil, fmt.Errorf("no default plan defined in operatorversion")
	}
	return plan, nil
}
