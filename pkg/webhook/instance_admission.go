package webhook

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/thoas/go-funk"
	"k8s.io/api/admission/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kudov1beta1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/controller/instance"
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
		// trigger "deploy" for freshly created Instances: req.Object contains the created object
		new := &kudov1beta1.Instance{}
		if err := ia.decoder.DecodeRaw(req.Object, new); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		new.Spec.PlanExecution.PlanName = kudov1beta1.DeployPlanName
		return admission.Allowed("")

	case v1beta1.Update:
		// call validation for Instance Updates
		old, new := &kudov1beta1.Instance{}, &kudov1beta1.Instance{}

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
		ov, err := instance.GetOperatorVersion(new, ia.client)
		if err != nil {
			log.Printf("InstanceAdmission: Error getting operatorVersion %s for instance %s/%s: %v", new.Spec.OperatorVersion.Name, new.Namespace, new.Name, err)
			admission.Errored(http.StatusInternalServerError, err)
		}

		triggered, err := validateUpdate(old, new, ov)
		if err != nil {
			return admission.Denied(err.Error())
		}

		// Populate Instance.PlanExecution with the plan triggered by param change
		if triggered != nil {
			new.Spec.PlanExecution.PlanName = *triggered
		}

		return admission.Allowed("")

	default:
		return admission.Allowed("")
	}
}

/*
 A coarse-grained set of compatibility rules between upgrades, directly triggered plans and parameter updates
 with the focus on when an update should be declined.
 --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
| HasScheduledPlan | ParameterUpdate | Upgrade | PlanOverride | PlanCancel | Allow |                                                     Description                                                     |
|------------------|-----------------|---------|--------------|------------|-------|---------------------------------------------------------------------------------------------------------------------|
| x                | x               |         |              |            | [No]  | Forbid parameter updates when a plan is scheduled unless the same plan is triggered (instance status will be reset) |
| x                |                 | x       |              |            | No    | Forbid upgrades if another plan is running                                                                          |
| x                |                 |         | x            |            | No    | Forbid plan overrides (for now)                                                                                     |
| x                |                 |         |              | x          | No    | Forbid plan cancellations (for now)                                                                                 |
| ---              |                 |         |              |            |       | ---                                                                                                                 |
|                  | x               |         | x            |            | No    | Forbid simultaneous parameter update and directly triggered plan                                                    |
|                  |                 | x       | x            |            | No    | Forbid simultaneous upgrades and directly triggered plans                                                           |
|                  | x               | x       |              |            | No*   | Forbid simultaneous upgrades and parameter updates                                                                  |
 --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
 *Note: simultaneous upgrade and parameter update is NOT allowed. However, there is a exception where new OV removes an existing
 parameter. Removing this parameter in the instance update would not count as parameter update (since there is no plan to trigger).

 For the complete set of rules, see the corresponding test.
*/

// validateUpdate takes in the old and new (updated) instance and returns a new plan that might
// be triggered based on the update and an error if the update is not valid. Return plan might be
// - <nil> when there is no change to an existing scheduled plan
// - '' empty string when an existing plan should be canceled (not implemented yet)
// - 'newPlan' some new plan that should be triggered
func validateUpdate(old, new *kudov1beta1.Instance, ov *kudov1beta1.OperatorVersion) (*string, error) {
	// PREREQUISITES:
	newPlan := new.Spec.PlanExecution.PlanName
	oldPlan := old.Spec.PlanExecution.PlanName
	newOvRef := new.Spec.OperatorVersion
	oldOvRef := old.Spec.OperatorVersion

	paramDiff := kudov1beta1.ParameterDiff(old.Spec.Parameters, new.Spec.Parameters)
	paramDefs := kudov1beta1.GetParamDefinitions(paramDiff, ov)
	triggeredPlan, err := triggeredPlan(paramDefs, ov)
	if err != nil {
		return nil, fmt.Errorf("failed to update Instance %s/%s: %v", old.Namespace, old.Name, err)
	}

	hasPlan := oldPlan != ""
	isParameterUpdate := triggeredPlan != nil
	isUpgrade := newOvRef != oldOvRef
	isNovelPlan := !hasPlan && newPlan != ""
	isPlanOverride := hasPlan && newPlan != "" && newPlan != oldPlan
	isPlanCancellation := hasPlan && newPlan == ""

	// Checking compatibility rules described in the table above:
	switch {
	case hasPlan && isParameterUpdate && *triggeredPlan != oldPlan:
		return nil, fmt.Errorf("failed to update Instance %s/%s: plan '%s' is scheduled (or running) and an update would trigger a different plan '%s'", old.Namespace, old.Name, oldPlan, newPlan)
	case isUpgrade && hasPlan:
		return nil, fmt.Errorf("failed to update Instance %s/%s: upgrade to new OperatorVersion %s while a plan '%s' is scheduled (or running) is not allowed", old.Namespace, old.Name, newOvRef, oldPlan)
	case isUpgrade && isNovelPlan:
		return nil, fmt.Errorf("failed to update Instance %s/%s: upgrade to new OperatorVersion %s and triggering new plan '%s' is not allowed", old.Namespace, old.Name, newOvRef, newPlan)
	case isPlanOverride:
		return nil, fmt.Errorf("failed to update Instance %s/%s: overriding currently scheduled (or running) plan '%s' with '%s' is not supported", old.Namespace, old.Name, oldPlan, newPlan)
	case isPlanCancellation:
		return nil, fmt.Errorf("failed to update Instance %s/%s: cancelling currently scheduled (or running) plan '%s' is not supported", old.Namespace, old.Name, oldPlan)
	case isParameterUpdate && isUpgrade:
		return nil, fmt.Errorf("failed to update Instance %s/%s: upgrade to new OperatorVersion %s together with a parameter update triggering '%s' is not allowed", old.Namespace, old.Name, newOvRef, *triggeredPlan)
	case isParameterUpdate && isNovelPlan:
		return nil, fmt.Errorf("failed to update Instance %s/%s: triggering one plan '%s' directly and through parameter update '%s' is not allowed", old.Namespace, old.Name, oldPlan, newPlan)
	case isParameterUpdate && isPlanOverride:
		return nil, fmt.Errorf("failed to update Instance %s/%s: updating parameters and triggering plan '%s' is not allowed", old.Namespace, old.Name, *triggeredPlan)
	}

	// Deciding which plan to trigger:
	switch {
	case isUpgrade:
		plan := kudov1beta1.SelectPlan([]string{kudov1beta1.UpgradePlanName, kudov1beta1.UpdatePlanName, kudov1beta1.DeployPlanName}, ov)
		if plan == nil {
			return nil, fmt.Errorf("failed to update Instance %s/%s: couldn't find any suitable plan that would be triggered by an OperatorVersion upgrade", old.Namespace, old.Name)
		}
		return plan, nil

	case isParameterUpdate:
		// if the same plan is triggered by the update, we clean the Instance.Status to effectively restart the plan
		if *triggeredPlan == oldPlan {
			if err := resetInstanceStatus(old); err != nil {
				return nil, fmt.Errorf("failed to update Instance %s/%s: %v", old.Namespace, old.Name, err)
			}
		}
		return triggeredPlan, nil

	case isNovelPlan:
		return &newPlan, nil
		// Implement plan overrides and cancellations below:
	}

	return triggeredPlan, nil
}

// resetInstanceStatus clears Instance.Status to effectively restart existing plan
func resetInstanceStatus(instance *kudov1beta1.Instance) error {
	// TODO (AD): implement
	return nil
}

// triggeredPlan determines what plan to run based on parameters that changed and the corresponding parameter trigger.
func triggeredPlan(params []kudov1beta1.Parameter, ov *kudov1beta1.OperatorVersion) (*string, error) {
	// If no parameters were changed, we return an empty string so no plan would be triggered
	if len(params) == 0 {
		return nil, nil
	}

	plans := make([]string, 0)
	for _, p := range params {
		if p.Trigger != "" && kudov1beta1.SelectPlan([]string{p.Trigger}, ov) != nil {
			plans = append(plans, p.Trigger)
		}
	}

	plans = funk.UniqString(plans)

	switch len(plans) {
	case 0:
		// no plan could be triggered since we do not force existence of the "deploy" plan in the operators
		fallback := kudov1beta1.SelectPlan([]string{kudov1beta1.UpdatePlanName, kudov1beta1.DeployPlanName}, ov)
		if fallback == nil {
			return nil, fmt.Errorf("couldn't find any plans that would be triggered by the update")
		}
		return fallback, nil
	case 1:
		return &plans[0], nil
	default:
		return nil, fmt.Errorf("triggering multiple plans: [%v] at once is not allowed", plans)
	}
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
