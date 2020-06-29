package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"

	"github.com/thoas/go-funk"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	kudov1beta1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
)

// +k8s:deepcopy-gen=false

// InstanceAdmission validates updates to an Instance, guarding from conflicting plan executions
type InstanceAdmission struct {
	client  client.Client
	decoder *admission.Decoder
}

func NewInstanceAdmission(cfg *rest.Config, s *runtime.Scheme) (*InstanceAdmission, error) {
	// client.New returns a new Client using the provided config and Options.
	// The returned client reads *and* writes directly from the server
	// (it doesn't use object caches). Using a cached client might lead to racy
	// behaviour when installing operators e.g. and `OperatorVersion` is already created
	// but not yet in cache which leads to an error during `Instance` creation.
	c, err := client.New(cfg, client.Options{Scheme: s})
	if err != nil {
		return nil, err
	}

	return &InstanceAdmission{client: c}, nil
}

// InstanceAdmission validates updates to an Instance, guarding from conflicting plan executions
func (ia *InstanceAdmission) Handle(ctx context.Context, req admission.Request) admission.Response {

	switch req.Operation {

	case v1beta1.Create:
		return handleCreate(ia, req)

	case v1beta1.Update:
		// req has both old and new Instance objects
		return handleUpdate(ia, req)

	default:
		return admission.Allowed("")
	}
}

func handleCreate(ia *InstanceAdmission, req admission.Request) admission.Response {
	// trigger "deploy" for freshly created Instances: req.Object contains the created object
	new := &kudov1beta1.Instance{}
	if err := ia.decoder.DecodeRaw(req.Object, new); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Metadata.Namespace of submitted objects is not trustworthy in Mutating Webhooks, and may be blank in Validation
	// Webhooks. The namespace needs to always be read out of the AdmissionReview object. For more information see:
	// https://github.com/kubernetes/kubernetes/issues/88282
	new.Namespace = req.Namespace

	// since we don't yet enforce the existence of the 'deploy' plan in the OV, we check for its existence
	// and decline Instance creation if the plan is not found
	ov, err := new.GetOperatorVersion(ia.client)
	if err != nil {
		log.Printf("InstanceAdmission: Error getting operatorVersion %s for instance %s/%s: %v", new.Spec.OperatorVersion.Name, new.Namespace, new.Name, err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// if there is a 'cleanup' plan present in the OV, we add a finalizer to the instance
	if kudov1beta1.CleanupPlanExists(ov) {
		new.TryAddFinalizer()
	}

	// schedule 'deploy' plan for execution (and fail if it doesn't exist)
	if !kudov1beta1.PlanExists(kudov1beta1.DeployPlanName, ov) {
		return admission.Denied(fmt.Sprintf("failed to create an Instance %s/%s: couldn't find '%s' plan in the operatorVersion", new.Namespace, new.Name, kudov1beta1.DeployPlanName))
	}
	new.Spec.PlanExecution.PlanName = kudov1beta1.DeployPlanName
	new.Spec.PlanExecution.UID = uuid.NewUUID()

	marshaled, err := json.Marshal(new)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaled)
}

func handleUpdate(ia *InstanceAdmission, req admission.Request) admission.Response {
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
	ov, err := new.GetOperatorVersion(ia.client)
	if err != nil {
		log.Printf("InstanceAdmission: Error getting operatorVersion %s for instance %s/%s: %v", new.Spec.OperatorVersion.Name, new.Namespace, new.Name, err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// we explicitly ignore Metadata updates
	if reflect.DeepEqual(old.Spec, new.Spec) && reflect.DeepEqual(old.Status, new.Status) {
		return admission.Allowed("")
	}

	triggered, err := admitUpdate(old, new, ov)
	if err != nil {
		return admission.Denied(err.Error())
	}

	// populate Instance.PlanExecution with the plan triggered by param change and evtl. a new UID/Status
	if triggered != nil {
		new.Spec.PlanExecution.PlanName = *triggered
		new.Spec.PlanExecution.UID = ""
		new.Spec.PlanExecution.Status = ""
		if *triggered != "" {
			new.Spec.PlanExecution.UID = uuid.NewUUID()                   // if there is a new plan, generate new UID
			new.Spec.PlanExecution.Status = kudov1beta1.ExecutionNeverRun // and set status to NEVER_RUN
		}

		marshaled, err := json.Marshal(new)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}

		return admission.PatchResponseFromRaw(req.Object.Raw, marshaled)
	}

	return admission.Allowed("")
}

/*
 A coarse-grained set of compatibility rules applied during the normal life-cycle phase of the Instance. Defines the rules applied for upgrades,
 directly triggered plans and parameter updates with the focus on when an update should be declined.
 --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
| HasScheduledPlan | ParameterUpdate | Upgrade | PlanOverride | PlanCancel | Allow |                                                     Description                                                     |
|------------------|-----------------|---------|--------------|------------|-------|---------------------------------------------------------------------------------------------------------------------|
| x                | x               |         |              |            | No²  | Forbid parameter updates when a plan is scheduled unless the same plan is triggered (instance status will be reset) |
| x                |                 | x       |              |            | No    | Forbid upgrades if another plan is running                                                                          |
| x                |                 |         | x            |            | No³   | Forbid plan overrides (for now)                                                                                     |
| x                |                 |         |              | x          | No    | Forbid plan cancellations (for now)                                                                                 |
| ---              |                 |         |              |            |       | ---                                                                                                                 |
|                  | x               |         | x            |            | No    | Forbid simultaneous parameter update and directly triggered plan                                                    |
|                  |                 | x       | x            |            | No    | Forbid simultaneous upgrades and directly triggered plans                                                           |
|                  | x               | x       |              |            | No*   | Forbid simultaneous upgrades and parameter updates                                                                  |
 --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
2. Simultaneous upgrade and parameter update are NOT allowed. However, there is a exception where new OV removes an existing
   parameter. Removing this parameter in the instance update would not count as parameter update (since there is no plan to
   trigger).
3. 'cleanup' plan is the only one that is allowed to override an existing one. Overriding 'cleanup' should be impossible even
   if an 'override=true' flag is introduced. This exception exists only during Instance cleanup phase.
*/

// admitUpdate takes in the old and new (updated) instance and returns a new plan that might
// be triggered based on the update and an error if the update is not valid. Return plan might be
// - <nil> when there is no change to an existing scheduled plan
// - '' empty string when an existing plan should be canceled (not implemented yet)
// - 'newPlan' some new plan that should be triggered
func admitUpdate(old, new *kudov1beta1.Instance, ov *kudov1beta1.OperatorVersion) (*string, error) { //nolint:gocyclo
	// PREREQUISITES:
	newPlan := new.Spec.PlanExecution.PlanName
	oldPlan := old.Spec.PlanExecution.PlanName
	newOvRef := new.Spec.OperatorVersion
	oldOvRef := old.Spec.OperatorVersion
	newUID := new.Spec.PlanExecution.UID
	oldUID := old.Spec.PlanExecution.UID

	// update and upgrade helpers
	hadPlan := oldPlan != ""
	isUpgrade := newOvRef != oldOvRef
	isNovelPlan := !hadPlan && newPlan != ""
	isPlanOverride := hadPlan && newPlan != "" && newPlan != oldPlan
	isPlanRetriggered := hadPlan && newPlan == oldPlan && newUID != oldUID
	isPlanCancellation := hadPlan && newPlan == ""
	isDeleting := new.IsDeleting() // a non-empty meta.deletionTimestamp is a signal to switch to the uninstalling life-cycle phase
	isPlanTerminal := new.Spec.PlanExecution.Status.IsTerminal()

	parameterDefs, err := changedParameterDefinitions(old.Spec.Parameters, new.Spec.Parameters, ov)
	if err != nil {
		return nil, fmt.Errorf("failed to update Instance %s/%s: %v", old.Namespace, old.Name, err)
	}

	triggeredPlan, err := triggeredByParameterUpdate(parameterDefs, ov)
	if err != nil {
		return nil, fmt.Errorf("failed to update Instance %s/%s: %v", old.Namespace, old.Name, err)
	}

	isParameterUpdate := triggeredPlan != nil
	updateIncompatibleWithUpgrade := isParameterUpdate && *triggeredPlan != kudov1beta1.DeployPlanName

	// --------------------------------------------------------------------------------------------
	// --- Instance can have two major life-cycle phases: normal and cleanup (uninstall) phase. ---
	// --- Different rule sets apply in both.                               				    ---
	// ---------------------------------------------------------------------------------------------

	// ----------------------------------
	// --- Instance uninstall/cleanup ---
	// ----------------------------------
	// Following rules apply:
	// - only 'cleanup' plan (if exists) can be scheduled in this phase
	// - only the instance controller (and not the webhook or the user) can schedule 'cleanup'
	// - 'cleanup' overrides any existing update/upgrade plan
	// - 'cleanup' itself can *NOT* be cancelled or overridden by any other plan since the instance is being deleted
	if isDeleting {
		Cleanup := kudov1beta1.CleanupPlanName
		isCleanupOverride := oldPlan == Cleanup && newPlan != oldPlan
		notCleanupScheduled := newPlan != "" && newPlan != Cleanup

		switch {
		case isCleanupOverride:
			return nil, fmt.Errorf("failed to update Instance %s/%s: '%s' plan can not be cancelled or overridden by another plan since the instance is being deleted", old.Namespace, old.Name, oldPlan)
		case isParameterUpdate || isUpgrade:
			return nil, fmt.Errorf("failed to update Instance %s/%s: parameter update and/or upgrade is not allowed when an instance is being deleted", old.Namespace, old.Name)
		case notCleanupScheduled:
			return nil, fmt.Errorf("failed to update Instance %s/%s: only '%s' plan can be scheduled when an instance is being deleted", old.Namespace, old.Name, Cleanup)
		}
		// cleanup is being scheduled by the controller so we don't have to return anything here
		return nil, nil
	}

	// ----------------------------
	// ---- Normal life-cycle -----
	// ----------------------------
	switch {
	case hadPlan && isParameterUpdate && *triggeredPlan != oldPlan:
		return nil, fmt.Errorf("failed to update Instance %s/%s: plan '%s' is scheduled (or running) and an update would trigger a different plan '%s'", old.Namespace, old.Name, oldPlan, *triggeredPlan)
	case isUpgrade && hadPlan:
		return nil, fmt.Errorf("failed to update Instance %s/%s: upgrade to new OperatorVersion %s while a plan '%s' is scheduled (or running) is not allowed", old.Namespace, old.Name, newOvRef, oldPlan)
	case isUpgrade && isNovelPlan:
		return nil, fmt.Errorf("failed to update Instance %s/%s: upgrade to new OperatorVersion %s and triggering new plan '%s' is not allowed", old.Namespace, old.Name, newOvRef, newPlan)
	case isUpgrade && updateIncompatibleWithUpgrade:
		return nil, fmt.Errorf("failed to update Instance %s/%s: upgrade to new OperatorVersion %s together with a parameter update triggering '%s' is not allowed", old.Namespace, old.Name, newOvRef, *triggeredPlan)
	case isPlanOverride:
		return nil, fmt.Errorf("failed to update Instance %s/%s: overriding currently scheduled (or running) plan '%s' with '%s' is not supported", old.Namespace, old.Name, oldPlan, newPlan)
	case isPlanCancellation:
		return nil, fmt.Errorf("failed to update Instance %s/%s: cancelling currently scheduled (or running) plan '%s' is not supported", old.Namespace, old.Name, oldPlan)
	case isParameterUpdate && isNovelPlan:
		return nil, fmt.Errorf("failed to update Instance %s/%s: triggering one plan '%s' directly and through parameter update '%s' is not allowed", old.Namespace, old.Name, oldPlan, newPlan)
	// this case is effectively a noop because isPlanOverride is disallowed for now. However, once plan overrides are implemented, this will be needed so don't remove.
	case isParameterUpdate && isPlanOverride:
		return nil, fmt.Errorf("failed to update Instance %s/%s: updating parameters and triggering plan '%s' is not allowed", old.Namespace, old.Name, *triggeredPlan)
	case newPlan == kudov1beta1.CleanupPlanName:
		return nil, fmt.Errorf("failed to update Instance %s/%s: only the controller schedules the '%s' plan when the instance is deleted", old.Namespace, old.Name, newPlan)
	}

	// Deciding which plan to trigger:
	switch {
	case isUpgrade:
		plan := kudov1beta1.SelectPlan([]string{kudov1beta1.UpgradePlanName, kudov1beta1.UpdatePlanName, kudov1beta1.DeployPlanName}, ov)
		if plan == nil {
			return nil, fmt.Errorf("failed to update Instance %s/%s: couldn't find any suitable plan that would be triggered by an OperatorVersion upgrade", old.Namespace, old.Name)
		}
		log.Printf("InstanceAdmission: instance %s/%s is being upgraded using %s plan", new.Namespace, new.Name, *plan)
		return plan, nil

	case isParameterUpdate:
		// if the same plan is triggered by the update, we clean the Instance.Status to effectively restart the plan
		log.Printf("InstanceAdmission: instance %s/%s, triggering %s plan after parameters has changed", new.Namespace, new.Name, *triggeredPlan)
		return triggeredPlan, nil

	case isNovelPlan:
		log.Printf("InstanceAdmission: instance %s/%s, new %s plan is triggered", new.Namespace, new.Name, newPlan)
		return &newPlan, nil

	case isPlanTerminal:
		// if current plan is terminal we reset the Instance.PlanExecution field and become ready for the new plan
		log.Printf("InstanceAdmission: instance %s/%s, %s plan is terminal", new.Namespace, new.Name, newPlan)
		empty := ""
		return &empty, nil

	case isPlanRetriggered:
		// return the existing plan which will lead to a new UID generated and hence the plan will be re-triggered
		log.Printf("InstanceAdmission: instance %s/%s, %s plan is re-triggered", new.Namespace, new.Name, newPlan)
		return &newPlan, nil

	default:
		// effectively nothing changed so it's a noop.
		log.Printf("InstanceAdmission: instance %s/%s no change in plan execution after the update", new.Namespace, new.Name)
		return nil, nil
	}
}

// triggeredByParameterUpdate determines what plan to run based on parameters that changed and the corresponding parameter trigger.
func triggeredByParameterUpdate(params []kudov1beta1.Parameter, ov *kudov1beta1.OperatorVersion) (*string, error) {
	// If no parameters were changed, we return an empty string so no plan would be triggered
	if len(params) == 0 {
		return nil, nil
	}

	plans := make([]string, 0)
	for _, p := range params {
		if p.Trigger != "" {
			if kudov1beta1.PlanExists(p.Trigger, ov) {
				plans = append(plans, p.Trigger)
			} else {
				return nil, fmt.Errorf("param %s defined trigger plan %s, but plan not defined in operatorversion", p.Name, p.Trigger)
			}
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

// merge method merges two maps and returns the result. Note, that left map is being modified in process.
func changedParameterDefinitions(old map[string]string, new map[string]string, ov *kudov1beta1.OperatorVersion) ([]kudov1beta1.Parameter, error) {
	c, r := kudov1beta1.RichParameterDiff(old, new)
	cpd, err := kudov1beta1.GetParamDefinitions(c, ov)
	if err != nil {
		return nil, err
	}

	// we ignore the error for missing OV parameter definitions for removed parameters. For once, this is a valid use-case when
	// upgrading an Instance (new OV might remove parameters), but the user can also manually edit current OV and remove parameters.
	// while discouraged, this is still possible since OV is not immutable.
	rpd, _ := kudov1beta1.GetParamDefinitions(r, ov)

	return append(cpd, rpd...), nil
}

// InstanceAdmission implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (ia *InstanceAdmission) InjectDecoder(d *admission.Decoder) error {
	ia.decoder = d
	return nil
}
