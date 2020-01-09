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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OperatorSpec defines the desired state of Operator
type OperatorSpec struct {
	Description       string
	KudoVersion       string
	KubernetesVersion string
	Maintainers       []*Maintainer
	URL               string
}

// Maintainer describes an Operator maintainer.
type Maintainer struct {
	// Name is a user name or organization name.
	Name string

	// Email is an optional email address to contact the named maintainer.
	Email string
}

// OperatorStatus defines the observed state of Operator
type OperatorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// Operator is the Schema for the operator API
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Operator struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec   OperatorSpec
	Status OperatorStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// OperatorList contains a list of Operator
type OperatorList struct {
	metav1.TypeMeta
	metav1.ListMeta
	Items []Operator
}
