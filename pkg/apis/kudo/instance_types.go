package kudo

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// Ensure conversion.Hub is implemented
var _ conversion.Hub = &Instance{}
var _ conversion.Hub = &InstanceList{}

// Instance is the Schema for the instances API.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Instance struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   InstanceSpec
	Status InstanceStatus
}

func (in *Instance) Hub() {}

// InstanceList contains a list of Instance.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type InstanceList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []Instance
}

func (in *InstanceList) Hub() {}

// InstanceSpec defines the desired state of Instance.
type InstanceSpec struct {
	// OperatorVersion specifies a reference to a specific OperatorVersion object.
	OperatorVersion corev1.ObjectReference

	Parameters v1beta1.JSON

	PlanExecution PlanExecution
}

type PlanExecution struct {
	PlanName string
	UID      apimachinerytypes.UID
	Status   ExecutionStatus
}

// InstanceStatus defines the observed state of Instance
type InstanceStatus struct {
	// slice would be enough here but we cannot use slice because order of sequence in yaml is considered significant while here it's not
	PlanStatus map[string]PlanStatus
}

type PlanStatus struct {
	Name    string
	Status  ExecutionStatus
	Message string
	// +nullable
	LastUpdatedTimestamp *metav1.Time
	Phases               []PhaseStatus
	UID                  apimachinerytypes.UID
}

// PhaseStatus is representing status of a phase
type PhaseStatus struct {
	Name    string
	Status  ExecutionStatus
	Message string
	Steps   []StepStatus
}

// StepStatus is representing status of a step
type StepStatus struct {
	Name    string
	Message string
	Status  ExecutionStatus
}

// ExecutionStatus captures the state of the rollout.
type ExecutionStatus string
