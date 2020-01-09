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

package kudo

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Instance struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   InstanceSpec
	Status InstanceStatus
}

// InstanceList contains a list of Instance.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type InstanceList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []Instance
}

// InstanceSpec defines the desired state of Instance.
type InstanceSpec struct {
	// OperatorVersion specifies a reference to a specific Operator object.
	OperatorVersion corev1.ObjectReference

	Parameters map[string]string
}

// InstanceStatus defines the observed state of Instance
type InstanceStatus struct {
	// slice would be enough here but we cannot use slice because order of sequence in yaml is considered significant while here it's not
	PlanStatus       map[string]PlanStatus
	AggregatedStatus AggregatedStatus
}

// AggregatedStatus is overview of an instance status derived from the plan status
type AggregatedStatus struct {
	Status         ExecutionStatus
	ActivePlanName string
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
	Name            string
	Status          ExecutionStatus
	Message         string // more verbose explanation of the status, e.g. a detailed error message
	LastFinishedRun metav1.Time
	Phases          []PhaseStatus
	UID             apimachinerytypes.UID
}

// PhaseStatus is representing status of a phase
type PhaseStatus struct {
	Name    string
	Status  ExecutionStatus
	Message string // more verbose explanation of the status, e.g. a detailed error message
	Steps   []StepStatus
}

// StepStatus is representing status of a step
type StepStatus struct {
	Name    string
	Message string // more verbose explanation of the status, e.g. a detailed error message
	Status  ExecutionStatus
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

	// CleanupPlanName is the name of the cleanup plan
	CleanupPlanName = "cleanup"
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

// InstanceError indicates error on that can also emit a kubernetes warn event
// +k8s:deepcopy-gen=false
// +k8s:conversion-gen=false
type InstanceError struct {
	Err       error
	EventName *string // nil if no warn event should be created
}

func (e *InstanceError) Error() string {
	return fmt.Sprintf("Error during execution: %v", e.Err)
}
