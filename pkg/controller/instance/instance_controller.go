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
	"reflect"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	"github.com/kudobuilder/kudo/pkg/kudoctl/resources/dependencies"
	"github.com/kudobuilder/kudo/pkg/util/convert"
)

// Reconciler reconciles an Instance object.
type Reconciler struct {
	client.Client
	Discovery discovery.CachedDiscoveryInterface
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
		// It is possible to filter out reconciling Instance.Status updates here by comparing
		// e.MetaNew.GetGeneration() != e.MetaOld.GetGeneration() for Instance resources. However, there is a pitfall
		// because a "nested operators" might install Instances and monitor their status. For more infos see:
		// https://github.com/kudobuilder/kudo/pull/1391
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
//   | Execute the scheduled plan    |
//   | if exists                     |
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

	log.Printf("InstanceController: Received Reconcile request for instance %s", request.NamespacedName)
	instance, err := r.getInstance(request)
	if err != nil {
		if apierrors.IsNotFound(err) { // not retrying if instance not found, probably someone manually removed it?
			log.Printf("Instance %s was deleted, nothing to reconcile.", request.NamespacedName)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	oldInstance := instance.DeepCopy()

	ov, err := instance.GetOperatorVersion(r.Client)
	if err != nil {
		err = fmt.Errorf("InstanceController: Error getting operatorVersion %s for instance %s/%s: %v",
			instance.Spec.OperatorVersion.Name, instance.Namespace, instance.Name, err)
		log.Print(err)
		r.Recorder.Event(instance, "Warning", "InvalidOperatorVersion", err.Error())
		return reconcile.Result{}, err // OV not found has to be retried because it can really have been created after Instance
	}

	// ---------- 2. Get currently scheduled plan if it exists ----------

	// get the scheduled plan
	plan, uid := scheduledPlan(instance, ov)
	if plan == "" {
		log.Printf("InstanceController: Nothing to do, no plan scheduled for instance %s/%s", instance.Namespace, instance.Name)
		return reconcile.Result{}, nil
	}

	ensurePlanStatusInitialized(instance, ov)

	// reset its status if the plan is new and log/record it
	planStatus, err := resetPlanStatusIfPlanIsNew(instance, plan, uid)
	if err != nil {
		log.Printf("InstanceController: Error resetting instance %s/%s status. %v", instance.Namespace, instance.Name, err)
		return reconcile.Result{}, err
	}

	if planStatus.Status == v1beta1.ExecutionPending {
		log.Printf("InstanceController: Going to start execution of plan '%s' on instance %s/%s", plan, instance.Namespace, instance.Name)
		r.Recorder.Event(instance, "Normal", "PlanStarted", fmt.Sprintf("Execution of plan %s started", plan))
	}

	// check if all the dependencies can be resolved (if necessary)
	err = r.resolveDependencies(instance, ov)
	if err != nil {
		planStatus.SetWithMessage(kudov1beta1.ExecutionFatalError, err.Error())
		instance.UpdateInstanceStatus(planStatus, &metav1.Time{Time: time.Now()})
		err = r.handleError(err, instance, oldInstance)
		return reconcile.Result{}, err
	}

	// ---------- 3. Execute the scheduled plan ----------

	metadata := &engine.Metadata{
		OperatorVersionName: ov.Name,
		OperatorVersion:     ov.Spec.Version,
		AppVersion:          ov.Spec.AppVersion,
		ResourcesOwner:      instance,
		OperatorName:        ov.Spec.Operator.Name,
		InstanceNamespace:   instance.Namespace,
		InstanceName:        instance.Name,
	}

	activePlan, err := preparePlanExecution(instance, ov, planStatus, metadata)
	if err != nil {
		err = r.handleError(err, instance, oldInstance)
		return reconcile.Result{}, err
	}
	log.Printf("InstanceController: Going to proceed with execution of the scheduled plan '%s' on instance %s/%s", activePlan.Name, instance.Namespace, instance.Name)
	newStatus, err := workflow.Execute(activePlan, metadata, r.Client, r.Discovery, r.Config, r.Scheme)

	// ---------- 4. Update instance and its status after the execution proceeded ----------

	if newStatus != nil {
		instance.UpdateInstanceStatus(newStatus, &metav1.Time{Time: time.Now()})
	}
	if err != nil {
		err = r.handleError(err, instance, oldInstance)
		return reconcile.Result{}, err
	}

	err = updateInstance(instance, oldInstance, r.Client)
	if err != nil {
		log.Printf("InstanceController: Error when updating instance %s/%s. %v", instance.Namespace, instance.Name, err)
		return reconcile.Result{}, err
	}

	// Publish a PlanFinished event after instance and its status were successfully updated
	if instance.Spec.PlanExecution.Status.IsTerminal() {
		r.Recorder.Event(instance, "Normal", "PlanFinished", fmt.Sprintf("Execution of plan %s finished with status %s", newStatus.Name, newStatus.Status))
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) resolveDependencies(i *kudov1beta1.Instance, ov *kudov1beta1.OperatorVersion) error {
	// no need to check the dependencies if this is a child-level instance, as the top-level instance will take care of that
	if i.IsChildInstance() {
		return nil
	}
	resolver := &InClusterResolver{ns: i.Namespace, c: r.Client}

	_, err := dependencies.Resolve(ov, resolver)
	if err != nil {
		return engine.ExecutionError{Err: fmt.Errorf("%w%v", engine.ErrFatalExecution, err), EventName: "CircularDependency"}
	}
	return nil
}

func updateInstance(instance *kudov1beta1.Instance, oldInstance *kudov1beta1.Instance, client client.Client) error {
	// The order of both updates below is important: *first* the instance Spec and Metadata and *then* the Status.
	// If Status is updated first, a new reconcile request will be scheduled and might fetch the *WRONG* instance
	// Spec.PlanExecution. This request will then try to execute an already finished plan (again).

	// 1. check if the finalizer can be removed (if the instance is being deleted and cleanup is completed) and then
	// update instance spec and metadata. this will not update Instance.Status field
	instance.TryRemoveFinalizer()

	if !reflect.DeepEqual(instance.Spec, oldInstance.Spec) ||
		!reflect.DeepEqual(instance.ObjectMeta, oldInstance.ObjectMeta) {

		instanceStatus := instance.Status.DeepCopy()

		err := client.Update(context.TODO(), instance)
		if err != nil {
			log.Printf("InstanceController: Error when updating instance spec. %v", err)
			return err
		}
		instance.Status = *instanceStatus
	}

	// 2. update instance status
	err := client.Status().Update(context.TODO(), instance)
	if err != nil {
		// if k8s GC was fast and managed to removed the instance (after the above Update removed the finalizer), we might get  an
		// untyped "StorageError" telling us that the sub-resource couldn't be modified. We ignore the error (but log it just in case).
		// historically we checked with a kerrors.IsNotFound() which failed based on the StorageError. Perhaps this is a k8s bug?
		if instance.IsDeleting() && instance.HasNoFinalizers() {
			log.Printf("InstanceController: failed status update for a deleted Instance %s/%s. (Ignored error: %v)", instance.Namespace, instance.Name, err)
			return nil
		}
		log.Printf("InstanceController: Error when updating instance status. %v", err)
		return err
	}

	return nil
}

func preparePlanExecution(instance *kudov1beta1.Instance, ov *kudov1beta1.OperatorVersion, activePlanStatus *kudov1beta1.PlanStatus, meta *engine.Metadata) (*workflow.ActivePlan, error) {
	planSpec, ok := ov.Spec.Plans[activePlanStatus.Name]
	if !ok {
		return nil, &engine.ExecutionError{Err: fmt.Errorf("%wcould not find required plan: '%v'", engine.ErrFatalExecution, activePlanStatus.Name), EventName: "InvalidPlan"}
	}

	params, err := ParamsMap(instance, ov)
	if err != nil {
		return nil, &engine.ExecutionError{Err: fmt.Errorf("%wcould not parse parameters: %v", engine.ErrFatalExecution, err), EventName: "InvalidParams"}
	}

	pipes, err := PipesMap(activePlanStatus.Name, &planSpec, ov.Spec.Tasks, meta)
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

	return err
}

// getInstance retrieves the instance by namespaced name
func (r *Reconciler) getInstance(request ctrl.Request) (instance *kudov1beta1.Instance, err error) {
	instance, err = kudov1beta1.GetInstance(request.NamespacedName, r.Client)
	if err != nil {
		log.Printf("InstanceController: Error getting instance %v: %v",
			request.NamespacedName,
			err)
		return nil, err
	}
	return instance, nil
}

// ParamsMap generates {{ Params.* }} map of keys and values which is later used during template rendering.
func ParamsMap(instance *kudov1beta1.Instance, operatorVersion *kudov1beta1.OperatorVersion) (map[string]interface{}, error) {
	params := make(map[string]interface{}, len(operatorVersion.Spec.Parameters))

	for _, param := range operatorVersion.Spec.Parameters {
		var value *string

		if v, ok := instance.Spec.Parameters[param.Name]; ok {
			value = &v
		} else {
			value = param.Default
		}

		var err error

		params[param.Name], err = convert.UnwrapParamValue(value, param.Type)
		if err != nil {
			return nil, err
		}
	}

	return params, nil
}

// PipesMap generates {{ Pipes.* }} map of keys and values which is later used during template rendering.
func PipesMap(planName string, plan *v1beta1.Plan, tasks []v1beta1.Task, emeta *engine.Metadata) (map[string]string, error) {
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

// resetPlanStatusIfPlanIsNew method resets a PlanStatus for a passed plan name and instance *IF* this is a newly
// scheduled plan (UID has changed) and returns updated plan status. In this case Plan/phase/step statuses are set
// to ExecutionPending meaning that the controller will restart plan execution. Otherwise (the plan is old),
// nothing is changed and the existing plan status is returned.
func resetPlanStatusIfPlanIsNew(i *v1beta1.Instance, plan string, uid types.UID) (*v1beta1.PlanStatus, error) {
	ps := i.PlanStatus(plan)
	if ps == nil {
		return nil, fmt.Errorf("failed to find planStatus for plan '%s'", plan)
	}

	// if plan UID is the same then we continue with the execution of the existing plan
	if ps.UID == uid {
		return ps, nil
	}

	// otherwise, we reset the plan phases and steps to ExecutionNeverRun
	i.ResetPlanStatus(ps, uid, &metav1.Time{Time: time.Now()})
	return ps, nil
}

// ensurePlanStatusInitialized initializes plan status for all plans this instance supports  it does not trigger run
// of any plan it either initializes everything for a fresh instance without any status or tries to adjust status
// after OV was updated
func ensurePlanStatusInitialized(i *v1beta1.Instance, ov *v1beta1.OperatorVersion) {
	if i.Status.PlanStatus == nil {
		i.Status.PlanStatus = make(map[string]v1beta1.PlanStatus)
	}

	for planName, plan := range ov.Spec.Plans {
		if _, ok := i.Status.PlanStatus[planName]; !ok {
			planStatus := v1beta1.PlanStatus{
				Name:   planName,
				Status: v1beta1.ExecutionNeverRun,
				Phases: make([]v1beta1.PhaseStatus, 0),
			}
			for _, phase := range plan.Phases {
				phaseStatus := v1beta1.PhaseStatus{
					Name:   phase.Name,
					Status: v1beta1.ExecutionNeverRun,
					Steps:  make([]v1beta1.StepStatus, 0),
				}
				for _, step := range phase.Steps {
					stepStatus := v1beta1.StepStatus{
						Name:   step.Name,
						Status: v1beta1.ExecutionNeverRun,
					}
					phaseStatus.Steps = append(phaseStatus.Steps, stepStatus)
				}
				planStatus.Phases = append(planStatus.Phases, phaseStatus)
			}
			i.Status.PlanStatus[planName] = planStatus
		}
	}
}

// scheduledPlan method returns currently scheduled plan and its UID from Instance.Spec.PlanExecution field. However, due
// to an edge case with instance deletion, this method also schedules the 'cleanup' plan if necessary (see the comments below)
func scheduledPlan(i *v1beta1.Instance, ov *v1beta1.OperatorVersion) (string, types.UID) {
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
		i.Spec.PlanExecution.Status = v1beta1.ExecutionNeverRun
	}

	return i.Spec.PlanExecution.PlanName, i.Spec.PlanExecution.UID
}
