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

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
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
	new := &kudoapi.Instance{}
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
	if kudoapi.CleanupPlanExists(ov) {
		new.TryAddFinalizer()
	}

	// schedule 'deploy' plan for execution (and fail if it doesn't exist)
	if !kudoapi.PlanExists(kudoapi.DeployPlanName, ov) {
		return admission.Denied(fmt.Sprintf("failed to create an Instance %s/%s: couldn't find '%s' plan in the operatorVersion", new.Namespace, new.Name, kudoapi.DeployPlanName))
	}
	new.Spec.PlanExecution.PlanName = kudoapi.DeployPlanName
	new.Spec.PlanExecution.UID = uuid.NewUUID()

	setImmutableParameterDefaults(ov, new)
	if err := validateParameters(ov, new); err != nil {
		return admission.Denied(fmt.Sprintf("failed to create an Instance %s/%s: parameters are not valid: %v", new.Namespace, new.Name, err))
	}

	marshaled, err := json.Marshal(new)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaled)
}

func handleUpdate(ia *InstanceAdmission, req admission.Request) admission.Response {
	old, new := &kudoapi.Instance{}, &kudoapi.Instance{}

	// req.Object contains the updated object
	if err := ia.decoder.DecodeRaw(req.Object, new); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// req.OldObject is populated for DELETE and UPDATE requests
	if err := ia.decoder.DecodeRaw(req.OldObject, old); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// we explicitly ignore Metadata updates
	if reflect.DeepEqual(old.Spec, new.Spec) && reflect.DeepEqual(old.Status, new.Status) {
		return admission.Allowed("")
	}

	// fetch new OperatorVersion: we always fetch the new one, since if it's an update it's the same as the old one
	// and if it's an upgrade, we need the new one anyway
	ov, err := new.GetOperatorVersion(ia.client)
	if err != nil {
		log.Printf("InstanceAdmission: Error getting operatorVersion %s for instance %s/%s: %v", new.Spec.OperatorVersion.Name, new.Namespace, new.Name, err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// we only need and use the oldOv if this is an upgrade to the OV, otherwise we don't need to fetch the
	// old one - it's the same as the current
	oldOv := ov
	if old.Spec.OperatorVersion != new.Spec.OperatorVersion {
		oldOv, err = old.GetOperatorVersion(ia.client)
		if err != nil {
			log.Printf("InstanceAdmission: Error getting operatorVersion %s for instance %s/%s: %v", old.Spec.OperatorVersion.Name, old.Namespace, old.Name, err)
			return admission.Errored(http.StatusInternalServerError, err)
		}
	}

	triggered, err := admitUpdate(old, new, ov, oldOv)
	if err != nil {
		return admission.Denied(err.Error())
	}

	// populate Instance.PlanExecution with the plan triggered by param change and evtl. a new UID/Status
	if triggered != nil {
		new.Spec.PlanExecution.PlanName = *triggered
		new.Spec.PlanExecution.UID = ""
		new.Spec.PlanExecution.Status = ""
		if *triggered != "" {
			new.Spec.PlanExecution.UID = uuid.NewUUID()               // if there is a new plan, generate new UID
			new.Spec.PlanExecution.Status = kudoapi.ExecutionNeverRun // and set status to NEVER_RUN
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
func admitUpdate(old, new *kudoapi.Instance, ov, oldOv *kudoapi.OperatorVersion) (*string, error) { //nolint:gocyclo
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

	// validate plan first
	if newPlan != "" && kudoapi.SelectPlan([]string{newPlan}, ov) == nil {
		return nil, fmt.Errorf("plan %s does not exist", newPlan)
	}

	changedDefs, removedDefs, err := changedParameters(old.Spec.Parameters, new.Spec.Parameters, oldOv, ov)
	if err != nil {
		return nil, fmt.Errorf("failed to update Instance %s/%s: %v", old.Namespace, old.Name, err)
	}

	if err = checkImmutableParameters(old.Spec.Parameters, new.Spec.Parameters, ov, oldOv, changedDefs, isUpgrade); err != nil {
		return nil, fmt.Errorf("failed to check immutable parameters for Instance %s/%s: %v", old.Namespace, old.Name, err)
	}
	if err := validateParameters(ov, new); err != nil {
		return nil, fmt.Errorf("failed to validate parameters for Instance %s/%s: %v", new.Namespace, new.Name, err)
	}

	updatedParameterDefs := append(changedDefs, removedDefs...)
	triggeredPlan, err := triggeredByParameterUpdate(updatedParameterDefs, ov)
	if err != nil {
		return nil, fmt.Errorf("failed to update Instance %s/%s: %v", old.Namespace, old.Name, err)
	}

	isParameterUpdate := triggeredPlan != nil
	updateIncompatibleWithUpgrade := isParameterUpdate && *triggeredPlan != kudoapi.DeployPlanName

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
		Cleanup := kudoapi.CleanupPlanName
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
	case newPlan == kudoapi.CleanupPlanName:
		return nil, fmt.Errorf("failed to update Instance %s/%s: only the controller schedules the '%s' plan when the instance is deleted", old.Namespace, old.Name, newPlan)
	}

	// Deciding which plan to trigger:
	switch {
	case isUpgrade:
		plan := kudoapi.SelectPlan([]string{kudoapi.UpgradePlanName, kudoapi.UpdatePlanName, kudoapi.DeployPlanName}, ov)
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
func triggeredByParameterUpdate(params []kudoapi.Parameter, ov *kudoapi.OperatorVersion) (*string, error) {
	// If no parameters were changed, we return an empty string so no plan would be triggered
	if len(params) == 0 {
		return nil, nil
	}

	plans := make([]string, 0)
	for _, p := range params {
		if p.Trigger != "" {
			if kudoapi.PlanExists(p.Trigger, ov) {
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
		fallback := kudoapi.SelectPlan([]string{kudoapi.UpdatePlanName, kudoapi.DeployPlanName}, ov)
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

// changedParameters returns a list of parameter definitions for params which value changed or that were added from old to new
// This does *not* include:
// - parameters which *definition* has changed in an OV upgrade but where the value has not changed
// - parameters that have been added in an OV upgrade
func changedParameters(old, new map[string]string, oldOv, newOv *kudoapi.OperatorVersion) ([]kudoapi.Parameter, []kudoapi.Parameter, error) {
	changedOrAdded, removed := kudoapi.RichParameterDiff(old, new)
	changedOrAddedDefs, err := kudoapi.GetParamDefinitions(changedOrAdded, newOv)
	if err != nil {
		return nil, nil, err
	}

	// Valid parameters not present in the old OV are not treated as changed.
	changedDefs := []kudoapi.Parameter{}
	for _, param := range changedOrAddedDefs {
		param := param
		if funk.Find(oldOv.Spec.Parameters, func(p kudoapi.Parameter) bool { return param.Name == p.Name }) != nil {
			changedDefs = append(changedDefs, param)
		}
	}

	// we ignore the error for missing OV parameter definitions for removed parameters. For once, this is a valid use-case when
	// upgrading an Instance (new OV might remove parameters), but the user can also manually edit current OV and remove parameters.
	// while discouraged, this is still possible since OV is not immutable.
	removedDefs, _ := kudoapi.GetParamDefinitions(removed, newOv)

	return changedDefs, removedDefs, nil
}

func checkImmutableParameters(old, new map[string]string, newOv, oldOv *kudoapi.OperatorVersion, changedDefs []kudoapi.Parameter, isUpgrade bool) error {
	if !isUpgrade {
		for _, p := range changedDefs {
			if p.IsImmutable() {
				return fmt.Errorf("parameter '%s' is immutable but was changed from '%v' to '%v'", p.Name, old[p.Name], new[p.Name])
			}
		}
	} else if err := validateOVUpgrade(new, newOv, oldOv); err != nil {
		return err
	}
	return nil
}

// validateOVUpgrade compares an old and a new OV with a set of new parameters and verifies that the upgrade operation is valid
//
// The rules are:
// Parameter is changed mutable -> immutable
//   * The value needs to be set in the given params list
// Parameter is changed immutable -> mutable
//   * Nothing to validate
// Immutable parameter is added to OV
//   * The value needs to be set in the given params list
// Immutable parameter is removed from OV
//   * Nothing to validate
//   * The value is removed from the params list
//
// https://github.com/kudobuilder/kudo/blob/main/keps/0030-immutable-parameters.md
func validateOVUpgrade(new map[string]string, newOv, oldOv *kudoapi.OperatorVersion) error {
	isEqualImmutable := func(p1, p2 kudoapi.Parameter) bool {
		return p1.IsImmutable() == p2.IsImmutable()
	}

	for _, changedParamDefs := range kudoapi.GetChangedParamDefs(oldOv.Spec.Parameters, newOv.Spec.Parameters, isEqualImmutable) {
		if changedParamDefs.IsImmutable() {
			// Param was changed from Mutable to Immutable - We need to make sure we have a value set
			if _, ok := new[changedParamDefs.Name]; !ok {
				return fmt.Errorf("parameter '%s' was changed to immutable in operator version %s but no value was provided", changedParamDefs.Name, newOv.Name)
			}
		}
		// else {
		// Param was changed from Immutable to Mutable - Nothing to validate
		// }
	}

	for _, removedParamDefs := range kudoapi.GetRemovedParamDefs(oldOv.Spec.Parameters, newOv.Spec.Parameters) {
		// Param was removed - Remove the value from the parameters if it exists
		delete(new, removedParamDefs.Name)
	}

	for _, addedParamDefs := range kudoapi.GetAddedParameters(oldOv.Spec.Parameters, newOv.Spec.Parameters) {
		if addedParamDefs.IsImmutable() {
			// A new immutable param was added
			if _, ok := new[addedParamDefs.Name]; !ok {
				if addedParamDefs.HasDefault() {
					return fmt.Errorf("parameter '%s' was added in operator version %s, but no value was provided (default would be %s)", addedParamDefs.Name, newOv.Name, *addedParamDefs.Default)
				}
				return fmt.Errorf("parameter '%s' was added in operator version %s, but no value was provided", addedParamDefs.Name, newOv.Name)
			}
		}
	}
	return nil
}

// validateParameters ensures that all parameters have correct values
func validateParameters(ov *kudoapi.OperatorVersion, instance *kudoapi.Instance) error {
	for _, p := range ov.Spec.Parameters {
		p := p
		pValue := instance.Spec.Parameters[p.Name]

		if err := p.ValidateValue(pValue); err != nil {
			return err
		}

	}
	return nil
}

// setImmutableParameterDefaults sets the default values for immutable parameters into the instances parameter map
func setImmutableParameterDefaults(ov *kudoapi.OperatorVersion, instance *kudoapi.Instance) {
	for _, p := range ov.Spec.Parameters {
		if p.IsImmutable() && p.HasDefault() {
			if instance.Spec.Parameters == nil {
				instance.Spec.Parameters = map[string]string{}
			}
			if _, ok := instance.Spec.Parameters[p.Name]; !ok {
				instance.Spec.Parameters[p.Name] = *p.Default
			}
		}
	}
}

// InstanceAdmission implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (ia *InstanceAdmission) InjectDecoder(d *admission.Decoder) error {
	ia.decoder = d
	return nil
}
