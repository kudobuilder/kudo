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

package v1beta1

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

// InstanceSpec defines the desired state of Instance.
type InstanceSpec struct {
	// OperatorVersion specifies a reference to a specific Operator object.
	OperatorVersion corev1.ObjectReference `json:"operatorVersion,omitempty"`

	Parameters map[string]string `json:"parameters,omitempty"`
}

// InstanceStatus defines the observed state of Instance
type InstanceStatus struct {
	// slice would be enough here but we cannot use slice because order of sequence in yaml is considered significant while here it's not
	PlanStatus       map[string]PlanStatus `json:"planStatus,omitempty"`
	AggregatedStatus AggregatedStatus      `json:"aggregatedStatus,omitempty"`
}

// AggregatedStatus is overview of an instance status derived from the plan status
type AggregatedStatus struct {
	Status         ExecutionStatus `json:"status,omitempty"`
	ActivePlanName string          `json:"activePlanName,omitempty"`
}

// PlanStatus is representing status of a plan
//
// These are valid states and transitions
//
//                       +----------------+
//                       | Never executed |
//                       +-------+--------+
//                               |
//                               v
//+-------------+        +-------+--------+
//|    Error    |<------>|    Pending     |
//+------+------+        +-------+--------+
//       ^                       |
//       |                       v
//       |               +-------+--------+
//       +-------------->|  In progress   |
//       |               +-------+--------+
//       |                       |
//       v                       v
//+------+------+        +-------+--------+
//| Fatal error |        |    Complete    |
//+-------------+        +----------------+
//
type PlanStatus struct {
	Name            string                `json:"name,omitempty"`
	Status          ExecutionStatus       `json:"status,omitempty"`
	Message         string                `json:"message,omitempty"` // more verbose explanation of the status, e.g. a detailed error message
	LastFinishedRun metav1.Time           `json:"lastFinishedRun,omitempty"`
	Phases          []PhaseStatus         `json:"phases,omitempty"`
	UID             apimachinerytypes.UID `json:"uid,omitempty"`
}

// PhaseStatus is representing status of a phase
type PhaseStatus struct {
	Name    string          `json:"name,omitempty"`
	Status  ExecutionStatus `json:"status,omitempty"`
	Message string          `json:"message,omitempty"` // more verbose explanation of the status, e.g. a detailed error message
	Steps   []StepStatus    `json:"steps,omitempty"`
}

// StepStatus is representing status of a step
type StepStatus struct {
	Name    string          `json:"name,omitempty"`
	Message string          `json:"message,omitempty"` // more verbose explanation of the status, e.g. a detailed error message
	Status  ExecutionStatus `json:"status,omitempty"`
}

func (s *StepStatus) Set(status ExecutionStatus) {
	s.Status = status
	s.Message = ""
}

func (s *StepStatus) SetWithMessage(status ExecutionStatus, message string) {
	s.Status = status
	s.Message = message
}

func (s *PhaseStatus) Set(status ExecutionStatus) {
	s.Status = status
	s.Message = ""
}

func (s *PhaseStatus) SetWithMessage(status ExecutionStatus, message string) {
	s.Status = status
	s.Message = message
}

func (s *PlanStatus) Set(status ExecutionStatus) {
	s.Status = status
	s.Message = ""
}

func (s *PlanStatus) SetWithMessage(status ExecutionStatus, message string) {
	s.Status = status
	s.Message = message
}

// ExecutionStatus captures the state of the rollout.
type ExecutionStatus string

const (
	// ExecutionInProgress actively deploying, but not yet healthy.
	ExecutionInProgress ExecutionStatus = "IN_PROGRESS"

	// ExecutionPending Not ready to deploy because dependent phases/steps not healthy.
	ExecutionPending ExecutionStatus = "PENDING"

	// ExecutionComplete deployed and healthy.
	ExecutionComplete ExecutionStatus = "COMPLETE"

	// ErrorStatus there was an error deploying the application.
	ErrorStatus ExecutionStatus = "ERROR"

	// ExecutionFatalError there was an error deploying the application.
	ExecutionFatalError ExecutionStatus = "FATAL_ERROR"

	// ExecutionNeverRun is used when this plan/phase/step was never run so far
	ExecutionNeverRun ExecutionStatus = "NEVER_RUN"

	// DeployPlanName is the name of the deployment plan
	DeployPlanName = "deploy"

	// UpgradePlanName is the name of the upgrade plan
	UpgradePlanName = "upgrade"

	// UpdatePlanName is the name of the update plan
	UpdatePlanName = "update"
)

// IsTerminal returns true if the status is terminal (either complete, or in a nonrecoverable error)
func (s ExecutionStatus) IsTerminal() bool {
	return s == ExecutionComplete || s == ExecutionFatalError
}

// IsFinished returns true if the status is complete regardless of errors
func (s ExecutionStatus) IsFinished() bool {
	return s == ExecutionComplete
}

// IsRunning returns true if the plan is currently being executed
func (s ExecutionStatus) IsRunning() bool {
	return s == ExecutionInProgress || s == ExecutionPending || s == ErrorStatus
}

// GetPlanInProgress returns plan status of currently active plan or nil if no plan is running
func (i *Instance) GetPlanInProgress() *PlanStatus {
	for _, p := range i.Status.PlanStatus {
		if p.Status.IsRunning() {
			return &p
		}
	}
	return nil
}

// GetLastExecutedPlanStatus returns status of plan that is currently running, if there is one running
// if no plan is running it looks for last executed plan based on timestamps
func (i *Instance) GetLastExecutedPlanStatus() *PlanStatus {
	if i.NoPlanEverExecuted() {
		return nil
	}
	activePlan := i.GetPlanInProgress()
	if activePlan != nil {
		return activePlan
	}
	var lastExecutedPlan *PlanStatus
	for n := range i.Status.PlanStatus {
		p := i.Status.PlanStatus[n]
		if p.Status == ExecutionNeverRun {
			continue // only interested in plans that run
		}
		if lastExecutedPlan == nil {
			lastExecutedPlan = &p // first plan that was run and we're iterating over
		} else if wasRunAfter(p, *lastExecutedPlan) {
			lastExecutedPlan = &p // this plan was run after the plan we have chosen before
		}
	}
	return lastExecutedPlan
}

// wasRunAfter returns true if p1 was run after p2
func wasRunAfter(p1 PlanStatus, p2 PlanStatus) bool {
	if p1.Status == ExecutionNeverRun || p2.Status == ExecutionNeverRun {
		return false
	}
	return p1.LastFinishedRun.Time.After(p2.LastFinishedRun.Time)
}

// NoPlanEverExecuted returns true is this is new instance for which we never executed any plan
func (i *Instance) NoPlanEverExecuted() bool {
	for _, p := range i.Status.PlanStatus {
		if p.Status != ExecutionNeverRun {
			return false
		}
	}
	return true
}

// EnsurePlanStatusInitialized initializes plan status for all plans this instance supports
// it does not trigger run of any plan
// it either initializes everything for a fresh instance without any status or tries to adjust status after OV was updated
func (i *Instance) EnsurePlanStatusInitialized(ov *OperatorVersion) {
	if i.Status.PlanStatus == nil {
		i.Status.PlanStatus = make(map[string]PlanStatus)
	}

	for planName, plan := range ov.Spec.Plans {
		planStatus := &PlanStatus{
			Name:   planName,
			Status: ExecutionNeverRun,
			Phases: make([]PhaseStatus, 0),
		}

		existingPlanStatus, planExists := i.Status.PlanStatus[planName]
		if planExists {
			planStatus.SetWithMessage(existingPlanStatus.Status, existingPlanStatus.Message)
		}
		for _, phase := range plan.Phases {
			phaseStatus := &PhaseStatus{
				Name:   phase.Name,
				Status: ExecutionNeverRun,
				Steps:  make([]StepStatus, 0),
			}
			existingPhaseStatus, phaseExists := PhaseStatus{}, false
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
				stepStatus := StepStatus{
					Name:   step.Name,
					Status: ExecutionNeverRun,
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

// StartPlanExecution mark plan as to be executed
// this modifies the instance.Status as well as instance.Metadata.Annotation (to save snapshot if needed)
func (i *Instance) StartPlanExecution(planName string, ov *OperatorVersion) error {
	if i.NoPlanEverExecuted() || isUpgradePlan(planName) {
		i.EnsurePlanStatusInitialized(ov)
	}

	// update status of the instance to reflect the newly starting plan
	notFound := true
	for planIndex, v := range i.Status.PlanStatus {
		if v.Name == planName {
			// update plan status
			notFound = false
			planStatus := i.Status.PlanStatus[planIndex]
			planStatus.Set(ExecutionPending)
			planStatus.UID = uuid.NewUUID()
			for j, p := range v.Phases {
				planStatus.Phases[j].Set(ExecutionPending)
				for k := range p.Steps {
					i.Status.PlanStatus[planIndex].Phases[j].Steps[k].Set(ExecutionPending)
				}
			}

			i.Status.PlanStatus[planIndex] = planStatus // we cannot modify item in map, we need to reassign here

			// update activePlan and instance status
			i.Status.AggregatedStatus.Status = ExecutionPending
			i.Status.AggregatedStatus.ActivePlanName = planName

			break
		}
	}
	if notFound {
		return &InstanceError{fmt.Errorf("asked to execute a plan %s but no such plan found in instance %s/%s", planName, i.Namespace, i.Name), kudo.String("PlanNotFound")}
	}

	err := i.SaveSnapshot()
	if err != nil {
		return err
	}

	return nil
}

// isUpgradePlan returns true if this could be an upgrade plan - this is just an approximation because deploy plan can be used for both
func isUpgradePlan(planName string) bool {
	return planName == DeployPlanName || planName == UpgradePlanName
}

// UpdateInstanceStatus updates `Status.PlanStatus` and `Status.AggregatedStatus` property based on the given plan
func (i *Instance) UpdateInstanceStatus(planStatus *PlanStatus) {
	for k, v := range i.Status.PlanStatus {
		if v.Name == planStatus.Name {
			i.Status.PlanStatus[k] = *planStatus
			i.Status.AggregatedStatus.Status = planStatus.Status
			if planStatus.Status.IsTerminal() {
				i.Status.AggregatedStatus.ActivePlanName = ""
			}
		}
	}
}

const snapshotAnnotation = "kudo.dev/last-applied-instance-state"

// SaveSnapshot stores the current spec of Instance into the snapshot annotation
// this information is used when executing update/upgrade plans, this overrides any snapshot that existed before
func (i *Instance) SaveSnapshot() error {
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

func (i *Instance) snapshotSpec() (*InstanceSpec, error) {
	if i.Annotations != nil {
		snapshot, ok := i.Annotations[snapshotAnnotation]
		if ok {
			var spec *InstanceSpec
			err := json.Unmarshal([]byte(snapshot), &spec)
			if err != nil {
				return nil, err
			}
			return spec, nil
		}
	}
	return nil, nil
}

// selectPlan returns nil if none of the plan exists, otherwise the first one in list that exists
func selectPlan(possiblePlans []string, ov *OperatorVersion) *string {
	for _, n := range possiblePlans {
		if _, ok := ov.Spec.Plans[n]; ok {
			return kudo.String(n)
		}
	}
	return nil
}

// GetPlanToBeExecuted returns name of the plan that should be executed
func (i *Instance) GetPlanToBeExecuted(ov *OperatorVersion) (*string, error) {
	if i.GetPlanInProgress() != nil { // we're already running some plan
		return nil, nil
	}

	// new instance, need to run deploy plan
	if i.NoPlanEverExecuted() {
		return kudo.String(DeployPlanName), nil
	}

	// did the instance change so that we need to run deploy/upgrade/update plan?
	instanceSnapshot, err := i.snapshotSpec()
	if err != nil {
		return nil, err
	}
	if instanceSnapshot == nil {
		// we don't have snapshot -> we never run deploy, also we cannot run update/upgrade. This should never happen
		return nil, &InstanceError{fmt.Errorf("unexpected state: no plan is running, no snapshot present - this should never happen :) for instance %s/%s", i.Namespace, i.Name), kudo.String("UnexpectedState")}
	}
	if instanceSnapshot.OperatorVersion.Name != i.Spec.OperatorVersion.Name {
		// this instance was upgraded to newer version
		log.Printf("Instance: instance %s/%s was upgraded from %s to %s operatorVersion", i.Namespace, i.Name, instanceSnapshot.OperatorVersion.Name, i.Spec.OperatorVersion.Name)
		plan := selectPlan([]string{UpgradePlanName, UpdatePlanName, DeployPlanName}, ov)
		if plan == nil {
			return nil, &InstanceError{fmt.Errorf("supposed to execute plan because instance %s/%s was upgraded but none of the deploy, upgrade, update plans found in linked operatorVersion", i.Namespace, i.Name), kudo.String("PlanNotFound")}
		}
		return plan, nil
	}
	// did instance parameters change, so that the corresponding plan has to be triggered?
	if !reflect.DeepEqual(instanceSnapshot.Parameters, i.Spec.Parameters) {
		// instance updated
		log.Printf("Instance: instance %s/%s has updated parameters from %v to %v", i.Namespace, i.Name, instanceSnapshot.Parameters, i.Spec.Parameters)
		paramDiff := parameterDifference(instanceSnapshot.Parameters, i.Spec.Parameters)
		paramDefinitions := getParamDefinitions(paramDiff, ov)
		plan := planNameFromParameters(paramDefinitions, ov)
		if plan == nil {
			return nil, &InstanceError{fmt.Errorf("supposed to execute plan because instance %s/%s was updated but none of the deploy, update plans found in linked operatorVersion", i.Namespace, i.Name), kudo.String("PlanNotFound")}
		}
		return plan, nil
	}
	return nil, nil
}

// planNameFromParameters determines what plan to run based on params that changed and the related trigger plans
func planNameFromParameters(params []Parameter, ov *OperatorVersion) *string {
	for _, p := range params {
		// TODO: if the params have different trigger plans, we always select first here which might not be ideal
		if p.Trigger != "" && selectPlan([]string{p.Trigger}, ov) != nil {
			return kudo.String(p.Trigger)
		}
	}
	return selectPlan([]string{UpdatePlanName, DeployPlanName}, ov)
}

// getParamDefinitions retrieves parameter metadata from OperatorVersion CRD
func getParamDefinitions(params map[string]string, ov *OperatorVersion) []Parameter {
	defs := []Parameter{}
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

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Instance is the Schema for the instances API.
// +k8s:openapi-gen=true
type Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanceSpec   `json:"spec,omitempty"`
	Status InstanceStatus `json:"status,omitempty"`
}

// OperatorVersionNamespace returns the namespace of the OperatorVersion that the Instance references.
func (i *Instance) OperatorVersionNamespace() string {
	if i.Spec.OperatorVersion.Namespace == "" {
		return i.ObjectMeta.Namespace
	}
	return i.Spec.OperatorVersion.Namespace
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InstanceList contains a list of Instance.
type InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Instance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Instance{}, &InstanceList{})
}

// InstanceError indicates error on that can also emit a kubernetes warn event
// +k8s:deepcopy-gen=false
type InstanceError struct {
	err       error
	EventName *string // nil if no warn event should be created
}

func (e *InstanceError) Error() string {
	return fmt.Sprintf("Error during execution: %v", e.err)
}
