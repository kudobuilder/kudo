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
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"

	"github.com/thoas/go-funk"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

const (
	snapshotAnnotation           = "kudo.dev/last-applied-instance-state"
	instanceCleanupFinalizerName = "kudo.dev.instance.cleanup"
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

	// ---------- 2. Check if the object is being deleted ----------

	if !instance.IsDeleting() {
		if _, hasCleanupPlan := ov.Spec.Plans[kudov1beta1.CleanupPlanName]; hasCleanupPlan {
			if tryAddFinalizer(instance) {
				log.Printf("InstanceController: Adding finalizer on instance %s/%s", instance.Namespace, instance.Name)
			}
		}
	} else {
		log.Printf("InstanceController: Instance %s/%s is being deleted", instance.Namespace, instance.Name)
	}

	// ---------- 3. Check if we should start execution of new plan ----------

	planToBeExecuted, err := getPlanToBeExecuted(instance, ov)
	if err != nil {
		return reconcile.Result{}, err
	}
	if planToBeExecuted != nil {
		log.Printf("InstanceController: Going to start execution of plan %s on instance %s/%s", kudo.StringValue(planToBeExecuted), instance.Namespace, instance.Name)
		err = startPlanExecution(instance, kudo.StringValue(planToBeExecuted), ov, time.Now())
		if err != nil {
			return reconcile.Result{}, r.handleError(err, instance, oldInstance)
		}
		r.Recorder.Event(instance, "Normal", "PlanStarted", fmt.Sprintf("Execution of plan %s started", kudo.StringValue(planToBeExecuted)))
	}

	// ---------- 4. If there's currently active plan, continue with the execution ----------

	activePlanStatus := instance.GetPlanInProgress()
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

	// ---------- 5. Update status of instance after the execution proceeded ----------
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
	if tryRemoveFinalizer(instance) {
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
func GetOperatorVersion(instance *kudov1beta1.Instance, c client.Client) (ov *kudov1beta1.OperatorVersion, err error) {
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
func startPlanExecution(i *v1beta1.Instance, planName string, ov *v1beta1.OperatorVersion, currentTime time.Time) error {
	if i.NoPlanEverExecuted() || isUpgradePlan(planName) {
		ensurePlanStatusInitialized(i, ov)
	}

	// update status of the instance to reflect the newly starting plan
	notFound := true
	for planIndex, v := range i.Status.PlanStatus {
		if v.Name == planName {
			// update plan status
			notFound = false
			planStatus := i.Status.PlanStatus[planIndex]
			planStatus.Set(v1beta1.ExecutionPending)
			planStatus.UID = uuid.NewUUID()
			for j, p := range v.Phases {
				planStatus.Phases[j].Set(v1beta1.ExecutionPending)
				for k := range p.Steps {
					i.Status.PlanStatus[planIndex].Phases[j].Steps[k].Set(v1beta1.ExecutionPending)
				}
			}

			i.Status.PlanStatus[planIndex] = planStatus // we cannot modify item in map, we need to reassign here

			// update activePlan and instance status
			i.Status.AggregatedStatus.Status = v1beta1.ExecutionPending
			i.Status.AggregatedStatus.ActivePlanName = planName
			i.Status.AggregatedStatus.LastUpdated = &v1.Time{Time: currentTime}

			break
		}
	}
	if notFound {
		return &v1beta1.InstanceError{Err: fmt.Errorf("asked to execute a plan %s but no such plan found in instance %s/%s", planName, i.Namespace, i.Name), EventName: kudo.String("PlanNotFound")}
	}

	err := saveSnapshot(i)
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

// getPlanToBeExecuted returns name of the plan that should be executed
func getPlanToBeExecuted(i *v1beta1.Instance, ov *v1beta1.OperatorVersion) (*string, error) {
	if i.IsDeleting() {
		// we have a cleanup plan
		plan := kudov1beta1.SelectPlan([]string{v1beta1.CleanupPlanName}, ov)
		if plan != nil {
			if planStatus := i.PlanStatus(*plan); planStatus != nil {
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

	if i.GetPlanInProgress() != nil { // we're already running some plan
		return nil, nil
	}

	// new instance, need to run deploy plan
	if i.NoPlanEverExecuted() {
		return kudo.String(v1beta1.DeployPlanName), nil
	}

	// did the instance change so that we need to run deploy/upgrade/update plan?
	instanceSnapshot, err := snapshotSpec(i)
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

// SaveSnapshot stores the current spec of Instance into the snapshot annotation
// this information is used when executing update/upgrade plans, this overrides any snapshot that existed before
func saveSnapshot(i *v1beta1.Instance) error {
	jsonBytes, err := json.Marshal(i.Spec)
	if err != nil {
		return err
	}
	if i.Annotations == nil {
		i.Annotations = make(map[string]string)
	}
	i.Annotations[snapshotAnnotation] = string(jsonBytes)
	return nil
}

func snapshotSpec(i *v1beta1.Instance) (*v1beta1.InstanceSpec, error) {
	if i.Annotations != nil {
		snapshot, ok := i.Annotations[snapshotAnnotation]
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

func remove(values []string, s string) (result []string) {
	for _, value := range values {
		if value == s {
			continue
		}
		result = append(result, value)
	}
	return
}

// tryAddFinalizer adds the cleanup finalizer to an instance if the finalizer
// hasn't been added yet, the instance has a cleanup plan and the cleanup plan
// didn't run yet. Returns true if the cleanup finalizer has been added.
func tryAddFinalizer(i *v1beta1.Instance) bool {
	if !funk.ContainsString(i.ObjectMeta.Finalizers, instanceCleanupFinalizerName) {
		if planStatus := i.PlanStatus(v1beta1.CleanupPlanName); planStatus != nil {
			// avoid adding a finalizer again if a reconciliation is requested
			// after it has just been removed but the instance isn't deleted yet
			if planStatus.Status == v1beta1.ExecutionNeverRun {
				i.ObjectMeta.Finalizers = append(i.ObjectMeta.Finalizers, instanceCleanupFinalizerName)
				return true
			}
		}
	}

	return false
}

// tryRemoveFinalizer removes the cleanup finalizer of an instance if it has
// been added, the instance has a cleanup plan and the cleanup plan completed.
// Returns true if the cleanup finalizer has been removed.
func tryRemoveFinalizer(i *v1beta1.Instance) bool {
	if funk.ContainsString(i.ObjectMeta.Finalizers, instanceCleanupFinalizerName) {
		if planStatus := i.PlanStatus(v1beta1.CleanupPlanName); planStatus != nil {
			if planStatus.Status.IsTerminal() {
				i.ObjectMeta.Finalizers = remove(i.ObjectMeta.Finalizers, instanceCleanupFinalizerName)
				return true
			}
		} else {
			// We have a finalizer but no cleanup plan. This could be due to an updated instance.
			// Let's remove the finalizer.
			i.ObjectMeta.Finalizers = remove(i.ObjectMeta.Finalizers, instanceCleanupFinalizerName)
			return true
		}
	}

	return false
}
