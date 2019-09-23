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
	"fmt"

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
	// TODO: save snapshot of instance
	// update status of the instance to reflect the newly startin plan
	notFound := true
	for i1, v := range i.Status.PlanStatus {
		if v.Name == planName {
			// update plan status
			notFound = false
			i.Status.PlanStatus[i1].Status = ExecutionPending
			for i2, p := range v.Phases {
				i.Status.PlanStatus[i1].Phases[i2].Status = ExecutionPending
				for i3, _ := range p.Steps {
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
