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
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PlanExecutionSpec defines the desired state of PlanExecution
type PlanExecutionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	PlanName string                 `json:"planName"`
	Instance corev1.ObjectReference `json:"instance"`
}

// PlanExecutionStatus defines the observed state of PlanExecution
type PlanExecutionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Name     string     `json:"name,omitempty"`
	Strategy Ordering   `json:"strategy,omitempty"`
	State    PhaseState `json:"state,omitempty"`
	//Phases maps a phase name to a Phase object
	Phases []PhaseStatus `json:"phases,omitempty"`
}

//PhaseStatus specifies the status of list of steps that contain Kubernetes objects.
type PhaseStatus struct {
	Name     string     `json:"name,omitempty"`
	Strategy Ordering   `json:"strategy,omitempty"`
	State    PhaseState `json:"state,omitempty"`
	//Steps maps a step name to a list of mustached kubernetes objects stored as a string
	Steps []StepStatus `json:"steps"`
}

//StepStatus shows the status of the Step
type StepStatus struct {
	Name  string     `json:"name,omitempty"`
	State PhaseState `json:"state,omitempty"`
	//Objects will be serialized for each instance as the params and defaults
	// are provided, but not serialized in the payload
	Objects []runtime.Object `json:"-"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PlanExecution is the Schema for the planexecutions API
// +k8s:openapi-gen=true
type PlanExecution struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlanExecutionSpec   `json:"spec,omitempty"`
	Status PlanExecutionStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PlanExecutionList contains a list of PlanExecution
type PlanExecutionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PlanExecution `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PlanExecution{}, &PlanExecutionList{})
}
