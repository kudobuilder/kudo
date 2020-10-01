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
)

// OperatorVersionSpec defines the desired state of OperatorVersion.
type OperatorVersionSpec struct {
	// +optional
	Operator   corev1.ObjectReference `json:"operator,omitempty"`
	Version    string                 `json:"version,omitempty"`
	AppVersion string                 `json:"appVersion,omitempty"`

	// Templates is a list of references to YAML templates located in the templates folder and later referenced from tasks.
	Templates map[string]string `json:"templates,omitempty"`
	// List of all tasks available in this OperatorVersion.
	Tasks []Task `json:"tasks,omitempty"`

	Parameters []Parameter `json:"parameters,omitempty"`

	// Plans maps a plan name to a plan.
	// +nullable
	Plans map[string]Plan `json:"plans,omitempty"`

	// ConnectionString defines a templated string that can be used to connect to an instance of the Operator.
	// +optional
	ConnectionString string `json:"connectionString,omitempty"`

	// UpgradableFrom lists all OperatorVersions that can upgrade to this OperatorVersion.
	UpgradableFrom []corev1.ObjectReference `json:"upgradableFrom,omitempty"`
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
	// +optional
	Strategy Ordering `json:"strategy"`
	// Phases maps a phase name to a Phase object.
	// +optional
	// +nullable
	Phases []Phase `json:"phases"`
}

// ParameterType specifies the type of a parameter value.
type ParameterType string

const (
	// StringValueType is used for parameter values that are provided as a string.
	StringValueType ParameterType = "string"

	// IntegerValueType is used for parameter values that describe an integral number without any fractional part
	IntegerValueType ParameterType = "integer"

	// NumberValueType is used for parameter values that describe any numeric value, with or without a fractional part
	NumberValueType ParameterType = "number"

	// BooleanValueType is used for parameter values that are "true" or "false"
	BooleanValueType ParameterType = "boolean"

	// ArrayValueType is used for parameter values that describe an array of values.
	ArrayValueType ParameterType = "array"

	// MapValueType is used for parameter values that describe a mapping type.
	MapValueType ParameterType = "map"
)

// Phase specifies a list of steps that contain Kubernetes objects.
type Phase struct {
	// +optional
	Name string `json:"name"`
	// +optional
	Strategy Ordering `json:"strategy"`

	// Steps maps a step name to a list of templated Kubernetes objects stored as a string.
	// +optional
	Steps []Step `json:"steps"`
}

// Step defines a specific set of operations that occur.
type Step struct {
	// +optional
	Name string `json:"name"`
	// +optional
	Tasks []string `json:"tasks"`
}

// Task is a global, polymorphic implementation of all publicly available tasks
type Task struct {
	// +optional
	Name string `json:"name"`
	// +optional
	Kind string `json:"kind"`
	// +optional
	Spec TaskSpec `json:"spec"`
}

// TaskSpec embeds all possible task specs. This allows us to avoid writing custom un/marshallers that would only parse
// certain fields depending on the task Kind. The downside of this approach is, that embedded types can not have fields
// with the same json names as it would become ambiguous for the default parser. We might revisit this approach in the
// future should this become an issue.
type TaskSpec struct {
	ResourceTaskSpec     `json:",inline"`
	DummyTaskSpec        `json:",inline"`
	PipeTaskSpec         `json:",inline"`
	ToggleTaskSpec       `json:",inline"`
	KudoOperatorTaskSpec `json:",inline"`
}

// ResourceTaskSpec is referencing a list of resources
type ResourceTaskSpec struct {
	// +optional
	// +nullable
	Resources []string `json:"resources,omitempty"`
}

// ToggleTaskSpec is referencing a ResourceTaskSpec and a parameter
type ToggleTaskSpec struct {
	// +optional
	Parameter string `json:"parameter,omitempty"`
}

// DummyTaskSpec can succeed or fail on demand and is very useful for testing operators
type DummyTaskSpec struct {
	// +optional
	WantErr bool `json:"wantErr,omitempty"`
	// +optional
	Fatal bool `json:"fatal,omitempty"`
	// +optional
	Done bool `json:"done,omitempty"`
}

// PipeTask specifies a task that generates files and stores them for later usage in subsequent tasks
type PipeTaskSpec struct {
	// +optional
	Pod string `json:"pod,omitempty"`
	// +optional
	// +nullable
	Pipe []PipeSpec `json:"pipe,omitempty"`
}

// PipeSpec describes how a file generated by a PipeTask is stored and referenced
type PipeSpec struct {
	// +optional
	File string `json:"file"`
	// +optional
	EnvFile string `json:"envFile"`
	// +optional
	Kind string `json:"kind"`
	// +optional
	Key string `json:"key"`
}

// KudoOperatorSpec specifies how a KUDO operator is installed
type KudoOperatorTaskSpec struct {
	// either repo package name, local package folder or an URL to package tarball. during operator installation,
	// kudoctl will resolve the package and override this field with the resolved operator name.
	// +optional
	Package string `json:"package,omitempty"`
	// +optional
	InstanceName string `json:"instanceName,omitempty"`
	// a specific app version in the official repo, defaults to the most recent
	// +optional
	AppVersion string `json:"appVersion,omitempty"`
	// a specific operator version in the official repo, defaults to the most recent one
	// +optional
	OperatorVersion string `json:"operatorVersion,omitempty"`
	// name of the template file (located in the `templates` folder) from which the *parent* instance
	// generates a parameter file used to populate the *child* Instance.Spec.Parameters
	// +optional
	ParameterFile string `json:"parameterFile,omitempty"`
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
