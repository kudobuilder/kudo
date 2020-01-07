package v1beta1

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"reflect"

	"github.com/thoas/go-funk"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +k8s:deepcopy-gen=false

// InstanceAdmission validates updates to an Instance, guarding from conflicting plan executions
type InstanceAdmission struct {
	client  client.Client
	decoder *admission.Decoder
}

// InstanceAdmission validates updates to an Instance, guarding from conflicting plan executions
func (ia *InstanceAdmission) Handle(ctx context.Context, req admission.Request) admission.Response {

	switch req.Operation {

	case v1beta1.Create:
		// 0. Trigger "deploy" by setting Instance.PlanExecution.PlanName = "deploy"
		return admission.Allowed("")
	// we only validate Instance Updates
	case v1beta1.Update:
		old, new := &Instance{}, &Instance{}

		// req.Object contains the updated object
		if err := ia.decoder.DecodeRaw(req.Object, new); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		// req.OldObject is populated for DELETE and UPDATE requests
		if err := ia.decoder.DecodeRaw(req.OldObject, old); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		// fetch new OperatorVersion: we always fetch the new one, since if it's an update it's the same as the old one
		// and if it's an upgrade, we need the new one anyway
		ov, err := ia.getOperatorVersion(new)
		if err != nil {
			admission.Errored(http.StatusInternalServerError, err)
		}

		triggered, err := validateUpdate(old, new, ov)
		if err != nil {
			return admission.Denied(err.Error())
		}

		// Populate Instance.PlanExecution with the plan triggered by param change
		new.Spec.PlanExecution.PlanName = triggered

		// PROFIT!
		return admission.Allowed("")
	default:
		return admission.Allowed("")
	}
}

func validateUpdate(old, new *Instance, ov *OperatorVersion) (string, error) {
	// Prereqs:
	//  a) new PE (Instance.Spec.PlanExecution.PlanName)
	//  b) new Params (parameterDiff)
	//  c) new OV (Instance.Spec.OperatorVersion)
	newPlan := new.Spec.PlanExecution.PlanName
	oldPlan := old.Spec.PlanExecution.PlanName
	paramDiff := parameterDiff(old.Spec.Parameters, new.Spec.Parameters)
	newOvRef := new.Spec.OperatorVersion
	oldOvRef := old.Spec.OperatorVersion

	// DECLINE if:
	// 1. old PE exists and != new PE : no plan overriding yet
	if oldPlan != "" && oldPlan != newPlan {
		return "", fmt.Errorf("failed to update Instance %s/%s: plan '%s' is scheduled and an update would trigger a different plan '%s'", old.Namespace, old.Name, oldPlan, newPlan)
	}

	// 2 OV changed and old PE exists : no upgrade if a plan running/scheduled
	if oldPlan != "" && newOvRef != oldOvRef {
		return "", fmt.Errorf("failed to update Instance %s/%s: upgrade to new OperatorVersion %s is not possible while a plan '%s' is scheduled", old.Namespace, old.Name, newOvRef, oldPlan)
	}

	// 3. OV changed and newPlan set : an upgrade should not be triggered with another plan
	if newPlan != "" && newOvRef != oldOvRef {
		return "", fmt.Errorf("failed to update Instance %s/%s: upgrade to new OperatorVersion %s is not possible together with a new plan '%s'", old.Namespace, old.Name, newOvRef, newPlan)
	}

	// 3.1 TODO (AD): No upgrade and no parameter diff detected : ???

	// 4. If >1 distinct plans are triggered based on params diff
	paramDefs := getParamDefinitions(paramDiff, ov)
	triggered, err := parameterDiffPlan(paramDefs, ov)
	if err != nil {
		return "", fmt.Errorf("failed to update Instance %s/%s: %v", old.Namespace, old.Name, err)
	}

	// 5. If old PE != plan triggered by param change
	if oldPlan != "" && triggered != oldPlan {
		return "", fmt.Errorf("failed to update Instance %s/%s: plan '%s' is scheduled and an update would trigger a different plan '%s'", old.Namespace, old.Name, oldPlan, triggered)
	}

	// if an already running plan is re-triggered by a parameter change, we reset the Instance.Status to
	// effectively restart the plan
	if oldPlan != "" && triggered == oldPlan {
		// TODO (av): reset the Instance.Status
	}

	// else ACCEPT and return the triggered plan
	return triggered, nil
}

// getOperatorVersion retrieves operator version belonging to the given instance
func (ia *InstanceAdmission) getOperatorVersion(instance *Instance) (ov *OperatorVersion, err error) {
	ov = &OperatorVersion{}
	err = ia.client.Get(context.TODO(),
		types.NamespacedName{
			Name:      instance.Spec.OperatorVersion.Name,
			Namespace: instance.OperatorVersionNamespace(),
		},
		ov)
	if err != nil {
		log.Printf("InstanceAdmission: Error getting operatorVersion %s for instance %s: %v", instance.Spec.OperatorVersion.Name, instance.Name, err)
		return nil, err
	}
	return ov, nil
}

// parameterDiffPlan determines what plan to run based on params that changed and the related trigger plans.
func parameterDiffPlan(params []Parameter, ov *OperatorVersion) (string, error) {
	// If no parameters were changed, we return an empty string so no plan would be triggered
	if len(params) == 0 {
		return "", nil
	}

	plans := make([]string, 0)
	for _, p := range params {
		if p.Trigger != "" && selectPlan([]string{p.Trigger}, ov) != nil {
			plans = append(plans, p.Trigger)
		}
	}

	plans = funk.UniqString(plans)

	switch len(plans) {
	case 0:
		// no plan could be triggered since we do not force existence of the "deploy" plan in the operators
		fallback := selectPlan([]string{UpdatePlanName, DeployPlanName}, ov)
		if fallback == nil {
			return "", fmt.Errorf("couldn't find any plans that would be triggered by the update")
		}
		return *fallback, nil
	case 1:
		return plans[0], nil
	default:
		return "", fmt.Errorf("triggering multiple plans: [%v] at once is not allowed", plans)
	}
}

func specChanged(old InstanceSpec, new InstanceSpec) bool {
	return !reflect.DeepEqual(old, new)
}

// InstanceAdmission implements inject.Client.
// A client will be automatically injected.

// InjectClient injects the client.
func (ia *InstanceAdmission) InjectClient(c client.Client) error {
	ia.client = c
	return nil
}

// InstanceAdmission implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (ia *InstanceAdmission) InjectDecoder(d *admission.Decoder) error {
	ia.decoder = d
	return nil
}
