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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// OperatorVersionSpec defines the desired state of OperatorVersion.
type OperatorVersionSpec struct {
	// +optional
	Operator corev1.ObjectReference `json:"operator,omitempty"`
	Version  string                 `json:"version,omitempty"`

	// Yaml captures a templated yaml list of elements that define the application operator instance.
	Templates map[string]string `json:"templates,omitempty"`
	Tasks     []Task            `json:"tasks,omitempty"`

	Parameters []Parameter `json:"parameters,omitempty"`

	// Plans maps a plan name to a plan.
	Plans map[string]Plan `json:"plans,omitempty"`

	// ConnectionString defines a templated string that can be used to connect to an instance of the Operator.
	// +optional
	ConnectionString string `json:"connectionString,omitempty"`

	// Dependencies a list of all dependencies of the operator.
	Dependencies []OperatorDependency `json:"dependencies,omitempty"`

	// UpgradableFrom lists all OperatorVersions that can upgrade to this OperatorVersion.
	UpgradableFrom []OperatorVersion `json:"upgradableFrom,omitempty"`

	AppVersion string `json:"appversion,omitempty"`
}

// Ordering specifies how the subitems in this plan/phase should be rolled out.
type Ordering string

const (
	// Serial specifies that the plans or objects should be created in order. The first should be healthy before
	// continuing on.
	Serial Ordering = "serial"

	// Parallel specifies that the plan or objects in the phase can all be launched at the same time.
	Parallel Ordering = "parallel"
)

// Plan specifies a series of Phases that need to be completed.
type Plan struct {
	Strategy Ordering `json:"strategy" validate:"required"` // makes field mandatory and checks if set and non empty
	// Phases maps a phase name to a Phase object.
	Phases []Phase `json:"phases" validate:"required,gt=0,dive"` // makes field mandatory and checks if its gt 0
}

// Parameter captures the variability of an OperatorVersion being instantiated in an instance.
type Parameter struct {
	// DisplayName can be used by UI's.
	DisplayName string `json:"displayName,omitempty"`

	// Name is the string that should be used in the templated file for example,
	// if `name: COUNT` then using the variable in a spec like:
	//
	// spec:
	//   replicas:  {{COUNT}}
	Name string `json:"name,omitempty"`

	// Description captures a longer description of how the parameter will be used.
	Description string `json:"description,omitempty"`

	// Required specifies if the parameter is required to be provided by all instances, or whether a default can suffice.
	Required bool `json:"required,omitempty"`

	// Default is a default value if no parameter is provided by the instance.
	Default *string `json:"default,omitempty"`

	// Trigger identifies the plan that gets executed when this parameter changes in the Instance object.
	// Default is `update` if a plan with that name exists, otherwise it's `deploy`
	Trigger string `json:"trigger,omitempty"`

	// TODO: Add generated parameters (e.g. passwords).
	// These values should be saved off in a secret instead of updating the spec
	// with values that viewing the instance does not return credentials.

}

// Phase specifies a list of steps that contain Kubernetes objects.
type Phase struct {
	Name     string   `json:"name" validate:"required"`     // makes field mandatory and checks if set and non empty
	Strategy Ordering `json:"strategy" validate:"required"` // makes field mandatory and checks if set and non empty

	// Steps maps a step name to a list of templated Kubernetes objects stored as a string.
	Steps []Step `json:"steps" validate:"required,gt=0,dive"` // makes field mandatory and checks if its gt 0
}

// Step defines a specific set of operations that occur.
type Step struct {
	Name   string   `json:"name" validate:"required"`            // makes field mandatory and checks if set and non empty
	Tasks  []string `json:"tasks" validate:"required,gt=0,dive"` // makes field mandatory and checks if non empty
	Delete bool     `json:"delete,omitempty"`                    // no checks needed

	// Objects will be serialized for each instance as the params and defaults are provided.
	Objects []runtime.Object `json:"-"` // no checks needed
}

// Task is a global, polymorphic implementation of all publicly available tasks
type Task struct {
	Name string   `json:"name" validate:"required"`
	Kind string   `json:"kind" validate:"required"`
	Spec TaskSpec `json:"spec" validate:"required"`
}

// TaskSpec embeds all possible task specs. This allows us to avoid writing custom un/marshallers that would only parse
// certain fields depending on the task Kind. The downside of this approach is, that embedded types can not have fields
// with the same json names as it would become ambiguous for the default parser. We might revisit this approach in the
// future should this become an issue.
type TaskSpec struct {
	ResourceTaskSpec
	DummyTaskSpec
}

// ResourceTaskSpec is referencing a list of resources
type ResourceTaskSpec struct {
	Resources []string `json:"resources"`
}

// DummyTaskSpec can succeed of fail on demand and is very useful for testing operators
type DummyTaskSpec struct {
	WantErr bool `json:"wantErr"`
	Fatal   bool `json:"fatal"`
	Done    bool `json:"done"`
}

// OperatorVersionStatus defines the observed state of OperatorVersion.
type OperatorVersionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OperatorVersion is the Schema for the operatorversions API.
// +k8s:openapi-gen=true
type OperatorVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OperatorVersionSpec   `json:"spec,omitempty"`
	Status OperatorVersionStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OperatorVersionList contains a list of OperatorVersion.
type OperatorVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OperatorVersion `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OperatorVersion{}, &OperatorVersionList{})
}

// OperatorDependency references a defined operator.
type OperatorDependency struct {
	// Name specifies the name of the dependency. Referenced via defaults.config.
	ReferenceName string `json:"referenceName"`
	corev1.ObjectReference

	// Version captures the requirements for what versions of the above object
	// are allowed.
	//
	// Example: ^3.1.4
	Version string `json:"version"`
}
