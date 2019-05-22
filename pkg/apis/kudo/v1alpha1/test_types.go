package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TestCase struct {
	// The type meta object, should always be a GVK of kudo.k8s.io/v1alpha1/TestCase.
	metav1.TypeMeta `json:",inline"`
	// Override the default metadata. Set labels or override the test case name.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Index int `json:"index,omitempty"`
	// Objects to delete at the beginning of the test case.
	Delete []corev1.ObjectReference `json:"delete,omitempty"`

	// Indicates that this is a unit test - safe to run without a real Kubernetes cluster.
	UnitTest bool `json:"unitTest"`

	// Allowed environment labels
	// Disallowed environment labels
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type TestAssert struct {
	// The type meta object, should always be a GVK of kudo.k8s.io/v1alpha1/TestAssert.
	metav1.TypeMeta `json:",inline"`
	// Override the default timeout of 300 seconds (in seconds).
	Timeout int `json:"timeout"`
}
