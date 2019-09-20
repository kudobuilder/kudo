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
	PlanStatus       map[string]PlanStatus `json:"planStatus,omitempty"`
	AggregatedStatus AggregatedStatus      `json:"aggregatedStatus,omitempty"`
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
	Status ExecutionStatus `json:"state,omitempty"`
	Steps  []StepStatus    `json:"steps,omitempty"`
}

// StepStatus is representing status of a step
type StepStatus struct {
	Name   string          `json:"name,omitempty"`
	Status ExecutionStatus `json:"state,omitempty"`
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

// ExecutionError there was an error deploying the application.
const ExecutionFatalError ExecutionStatus = "FATAL_ERROR"

func (s ExecutionStatus) IsTerminal() bool {
	return s == ExecutionComplete || s == ExecutionFatalError
}

func (s ExecutionStatus) IsRunning() bool {
	return s == ExecutionInProgress || s == ExecutionPending || s == ExecutionError
}

func (i *Instance) GetPlanInProgress() *PlanStatus {
	for _, p := range i.Status.PlanStatus {
		if p.Status.IsRunning() {
			return &p
		}
	}
	return nil
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
