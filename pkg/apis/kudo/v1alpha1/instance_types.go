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

package v1alpha1

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/kudobuilder/kudo/pkg/util/kudo"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstanceSpec defines the desired state of Instance.
type InstanceSpec struct {
	// OperatorVersion specifies a reference to a specific Operator object.
	OperatorVersion corev1.ObjectReference `json:"operatorVersion,omitempty"`

	Parameters map[string]string `json:"parameters,omitempty"`
}

// InstanceStatus defines the observed state of Instance
type InstanceStatus struct {
	PlanStatus       []PlanStatus     `json:"planStatus,omitempty"`
	AggregatedStatus AggregatedStatus `json:"aggregatedStatus,omitempty"`
}

// AggregatedStatus is overview of an instance status derived from the plan status
type AggregatedStatus struct {
	Status         ExecutionStatus `json:"status,omitempty"`
	ActivePlanName string          `json:"activePlanName,omitempty"`
}

// PlanStatus is representing status of a plan
type PlanStatus struct {
	Name            string          `json:"name,omitempty"`
	Status          ExecutionStatus `json:"status,omitempty"`
	LastFinishedRun metav1.Time     `json:"lastFinishedRun,omitempty"`
	Phases          []PhaseStatus   `json:"phases,omitempty"`
}

// PhaseStatus is representing status of a phase
type PhaseStatus struct {
	Name   string          `json:"name,omitempty"`
	Status ExecutionStatus `json:"status,omitempty"`
	Steps  []StepStatus    `json:"steps,omitempty"`
}

// StepStatus is representing status of a step
type StepStatus struct {
	Name   string          `json:"name,omitempty"`
	Status ExecutionStatus `json:"status,omitempty"`
}

// ExecutionStatus captures the state of the rollout.
type ExecutionStatus string

// ExecutionInProgress actively deploying, but not yet healthy.
const ExecutionInProgress ExecutionStatus = "IN_PROGRESS"

// ExecutionPending Not ready to deploy because dependent phases/steps not healthy.
const ExecutionPending ExecutionStatus = "PENDING"

// ExecutionComplete deployed and healthy.
const ExecutionComplete ExecutionStatus = "COMPLETE"

// ExecutionError there was an error deploying the application.
const ExecutionError ExecutionStatus = "ERROR"

// ExecutionFatalError there was an error deploying the application.
const ExecutionFatalError ExecutionStatus = "FATAL_ERROR"

// ExecutionNeverRun is used when this plan/phase/step was never run so far
const ExecutionNeverRun ExecutionStatus = "NEVER_RUN"

// IsTerminal returns true if the status is terminal (either complete, or in a nonrecoverable error)
func (s ExecutionStatus) IsTerminal() bool {
	return s == ExecutionComplete || s == ExecutionFatalError
}

// IsRunning returns true if the plan is currently being executed
func (s ExecutionStatus) IsRunning() bool {
	return s == ExecutionInProgress || s == ExecutionPending || s == ExecutionError
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
func (i *Instance) EnsurePlanStatusInitialized(ov *OperatorVersion) {
	i.Status.PlanStatus = make([]PlanStatus, 0)

	for planName, plan := range ov.Spec.Plans {
		planStatus := &PlanStatus{
			Name:   planName,
			Status: ExecutionNeverRun,
			Phases: make([]PhaseStatus, 0),
		}
		for _, phase := range plan.Phases {
			phaseStatus := &PhaseStatus{
				Name:   phase.Name,
				Status: ExecutionNeverRun,
				Steps:  make([]StepStatus, 0),
			}
			for _, step := range phase.Steps {
				phaseStatus.Steps = append(phaseStatus.Steps, StepStatus{
					Name:   step.Name,
					Status: ExecutionNeverRun,
				})
			}
			planStatus.Phases = append(planStatus.Phases, *phaseStatus)
		}
		i.Status.PlanStatus = append(i.Status.PlanStatus, *planStatus)
	}
}

// StartPlanExecution mark plan as to be executed
func (i *Instance) StartPlanExecution(planName string, ov *OperatorVersion) error {
	if i.NoPlanEverExecuted() {
		i.EnsurePlanStatusInitialized(ov)
	}

	// update status of the instance to reflect the newly starting plan
	notFound := true
	for i1, v := range i.Status.PlanStatus {
		if v.Name == planName {
			// update plan status
			notFound = false
			i.Status.PlanStatus[i1].Status = ExecutionPending
			for i2, p := range v.Phases {
				i.Status.PlanStatus[i1].Phases[i2].Status = ExecutionPending
				for i3 := range p.Steps {
					i.Status.PlanStatus[i1].Phases[i2].Steps[i3].Status = ExecutionPending
				}
			}

			// update activePlan and instance status
			i.Status.AggregatedStatus.Status = ExecutionPending
			i.Status.AggregatedStatus.ActivePlanName = planName

			break
		}
	}
	if notFound {
		return fmt.Errorf("asked to execute a plan %s but no such plan found in instance %s/%s", planName, i.Namespace, i.Name)
	}

	// save snapshot prior to execution for plans requiring snapshot
	if planName == "deploy" || planName == "update" || planName == "upgrade" {
		err := i.SaveSnapshot()
		if err != nil {
			return err
		}
	}

	return nil
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

func (i *Instance) getSnapshotedSpec() (*InstanceSpec, error) {
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
		return kudo.String("deploy"), nil
	}

	// did the instance change so that we need to run deploy/upgrade/update plan?
	instanceSnapshot, err := i.getSnapshotedSpec()
	if err != nil {
		return nil, err
	}
	if instanceSnapshot.OperatorVersion.Name != i.Spec.OperatorVersion.Name || instanceSnapshot.OperatorVersion.Namespace != i.Spec.OperatorVersion.Namespace {
		// this instance was upgraded to newer version
		plan := selectPlan([]string{"upgrade", "update", "deploy"}, ov)
		if plan == nil {
			return nil, fmt.Errorf("supposed to execute plan because instance %s/%s was upgraded but none of the deploy, upgrade, update plans found in linked operatorVersion", i.Namespace, i.Name)
		}
		return plan, nil
	}
	if reflect.DeepEqual(instanceSnapshot.Parameters, i.Spec.Parameters) {
		// instance updated
		paramDiff := parameterDifference(instanceSnapshot.Parameters, i.Spec.Parameters)
		paramDefinitions := getParamDefinitions(paramDiff, ov)
		plan := planNameFromParameters(paramDefinitions, ov)
		if plan == nil {
			return nil, fmt.Errorf("supposed to execute plan because instance %s/%s was updatet but none of the deploy, update plans found in linked operatorVersion", i.Namespace, i.Name)
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
	return selectPlan([]string{"update", "deploy"}, ov)
}

// getParamDefinitions retrieves parameter metadata from OperatorVersion CRD
func getParamDefinitions(params map[string]string, ov *OperatorVersion) []Parameter {
	defs := make([]Parameter, 0)
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

// GetOperatorVersionNamespace returns the namespace of the OperatorVersion that the Instance references.
func (i *Instance) GetOperatorVersionNamespace() string {
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
