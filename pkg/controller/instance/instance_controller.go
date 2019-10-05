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
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kudobuilder/kudo/pkg/util/kudo"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler reconciles an Instance object.
type Reconciler struct {
	client.Client
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
}

// SetupWithManager registers this reconciler with the controller manager
func (r *Reconciler) SetupWithManager(
	mgr ctrl.Manager) error {
	addOvRelatedInstancesToReconcile := handler.ToRequestsFunc(
		func(obj handler.MapObject) []reconcile.Request {
			requests := make([]reconcile.Request, 0)
			instances := &kudov1alpha1.InstanceList{}
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
		For(&kudov1alpha1.Instance{}).
		Owns(&kudov1alpha1.Instance{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&batchv1.Job{}).
		Owns(&appsv1.StatefulSet{}).
		Watches(&source.Kind{Type: &kudov1alpha1.OperatorVersion{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: addOvRelatedInstancesToReconcile}).
		Complete(r)
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
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kudo.dev,resources=instances,verbs=get;list;watch;create;update;patch;delete
func (r *Reconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	// ---------- 1. Query the current state ----------

	log.Printf("InstanceController: Received Reconcile request for instance \"%+v\"", request.Name)
	instance, err := r.getInstance(request)
	if err != nil {
		if apierrors.IsNotFound(err) { // not retrying if instance not found, probably someone manually removed it?
			log.Printf("Instances in namespace %s not found, not retrying reconcile since this error is usually not recoverable (without manual intervention).", request.NamespacedName)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	ov, err := r.getOperatorVersion(instance)
	if err != nil {
		return reconcile.Result{}, err // OV not found has to be retried because it can really have been created after Instance
	}

	// ---------- 2. First check if we should start execution of new plan ----------

	planToBeExecuted, err := instance.GetPlanToBeExecuted(ov)
	if err != nil {
		return reconcile.Result{}, err
	}
	if planToBeExecuted != nil {
		log.Printf("InstanceController: Going to start execution of plan %s on instance %s/%s", kudo.StringValue(planToBeExecuted), instance.Namespace, instance.Name)
		err = instance.StartPlanExecution(kudo.StringValue(planToBeExecuted), ov)
		if err != nil {
			return reconcile.Result{}, r.handleError(err, instance)
		}
		r.Recorder.Event(instance, "Normal", "PlanStarted", fmt.Sprintf("Execution of plan %s started", kudo.StringValue(planToBeExecuted)))
	}

	// ---------- 3. If there's currently active plan, continue with the execution ----------

	activePlanStatus := instance.GetPlanInProgress()
	if activePlanStatus == nil { // we have no plan in progress
		log.Printf("InstanceController: Nothing to do, no plan in progress for instance %s/%s", instance.Namespace, instance.Name)
		return reconcile.Result{}, nil
	}

	activePlan, metadata, err := preparePlanExecution(instance, ov, activePlanStatus)
	if err != nil {
		err = r.handleError(err, instance)
		return reconcile.Result{}, err
	}
	log.Printf("InstanceController: Going to proceed in execution of active plan %s on instance %s/%s", activePlan.Name, instance.Namespace, instance.Name)
	newStatus, err := executePlan(activePlan, metadata, r.Client, &kustomizeEnhancer{r.Scheme})

	// ---------- 4. Update status of instance after the execution proceeded ----------

	if newStatus != nil {
		instance.UpdateInstanceStatus(newStatus)
	}
	if err != nil {
		err = r.handleError(err, instance)
		return reconcile.Result{}, err
	}

	err = r.Client.Update(context.TODO(), instance)
	if err != nil {
		log.Printf("InstanceController: Error when updating instance state. %v", err)
		return reconcile.Result{}, err
	}

	if instance.Status.AggregatedStatus.Status.IsTerminal() {
		r.Recorder.Event(instance, "Normal", "PlanFinished", fmt.Sprintf("Execution of plan %s finished with status %s", activePlanStatus.Name, instance.Status.AggregatedStatus.Status))
	}

	return reconcile.Result{}, nil
}

func preparePlanExecution(instance *kudov1alpha1.Instance, ov *kudov1alpha1.OperatorVersion, activePlanStatus *kudov1alpha1.PlanStatus) (*activePlan, *executionMetadata, error) {
	params, err := getParameters(instance, ov)
	if err != nil {
		return nil, nil, err
	}

	planSpec, ok := ov.Spec.Plans[activePlanStatus.Name]
	if !ok {
		return nil, nil, &executionError{fmt.Errorf("could not find required plan (%v)", activePlanStatus.Name), false, kudo.String("InvalidPlan")}
	}

	return &activePlan{
			Name:       activePlanStatus.Name,
			Spec:       &planSpec,
			PlanStatus: activePlanStatus,
			Tasks:      ov.Spec.Tasks,
			Templates:  ov.Spec.Templates,
			params:     params,
		}, &executionMetadata{
			operatorVersionName: ov.Name,
			operatorVersion:     ov.Spec.Version,
			resourcesOwner:      instance,
			operatorName:        ov.Spec.Operator.Name,
			instanceNamespace:   instance.Namespace,
			instanceName:        instance.Name,
			appVersion:          ov.Spec.AppVersion,
		}, nil
}

// handleError handles execution error by logging, updating the plan status and optionally publishing an event
// specify eventReason as nil if you don't wish to publish a warning event
// returns err if this err should be retried, nil otherwise
func (r *Reconciler) handleError(err error, instance *kudov1alpha1.Instance) error {
	log.Printf("InstanceController: %v", err)

	// first update instance as we want to propagate errors also to the `Instance.Status.PlanStatus`
	clientErr := r.Client.Update(context.TODO(), instance)
	if clientErr != nil {
		log.Printf("InstanceController: Error when updating instance state. %v", clientErr)
		return clientErr
	}

	// determine if retry is necessary based on the error type
	if exErr, ok := err.(*executionError); ok {
		if exErr.eventName != nil {
			r.Recorder.Event(instance, "Warning", kudo.StringValue(exErr.eventName), err.Error())
		}

		if exErr.fatal {
			return nil // not retrying fatal error
		}
	}

	// for code being processed on instance, we need to handle these errors as well
	var iError *kudov1alpha1.InstanceError
	if errors.As(err, &iError) {
		if iError.EventName != nil {
			r.Recorder.Event(instance, "Warning", kudo.StringValue(iError.EventName), err.Error())
		}
	}
	return err
}

// getInstance retrieves the instance by namespaced name
// returns nil, nil when instance is not found (not found is not considered an error)
func (r *Reconciler) getInstance(request ctrl.Request) (instance *kudov1alpha1.Instance, err error) {
	instance = &kudov1alpha1.Instance{}
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

// getOperatorVersion retrieves operatorversion belonging to the given instance
// not found is treated here as any other error
func (r *Reconciler) getOperatorVersion(instance *kudov1alpha1.Instance) (ov *kudov1alpha1.OperatorVersion, err error) {
	ov = &kudov1alpha1.OperatorVersion{}
	err = r.Get(context.TODO(),
		types.NamespacedName{
			Name:      instance.Spec.OperatorVersion.Name,
			Namespace: instance.OperatorVersionNamespace(),
		},
		ov)
	if err != nil {
		log.Printf("InstanceController: Error getting operatorversion \"%v\" for instance \"%v\": %v",
			instance.Spec.OperatorVersion.Name,
			instance.Name,
			err)
		r.Recorder.Event(instance, "Warning", "InvalidOperatorVersion", fmt.Sprintf("Error getting operatorversion \"%v\": %v", instance.Spec.OperatorVersion.Name, err))
		return nil, err
	}
	return ov, nil
}

func getParameters(instance *kudov1alpha1.Instance, operatorVersion *kudov1alpha1.OperatorVersion) (map[string]string, error) {
	params := make(map[string]string)

	for k, v := range instance.Spec.Parameters {
		params[k] = v
	}

	missingRequiredParameters := make([]string, 0)
	// Merge defaults with customizations
	for _, param := range operatorVersion.Spec.Parameters {
		_, ok := params[param.Name]
		if !ok && param.Required && param.Default == nil {
			// instance does not define this parameter and there is no default while the parameter is required -> error
			missingRequiredParameters = append(missingRequiredParameters, param.Name)

		} else if !ok {
			params[param.Name] = kudo.StringValue(param.Default)
		}
	}

	if len(missingRequiredParameters) != 0 {
		return nil, &executionError{err: fmt.Errorf("parameters are missing when evaluating template: %s", strings.Join(missingRequiredParameters, ",")), fatal: true, eventName: kudo.String("Missing parameter")}
	}

	return params, nil
}

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

type executionError struct {
	err       error
	fatal     bool    // these errors should not be retried
	eventName *string // nil if no warn even should be created
}

func (e *executionError) Error() string {
	if e.fatal {
		return fmt.Sprintf("Fatal error: %v", e.err)
	}
	return fmt.Sprintf("Error during execution: %v", e.err)
}
