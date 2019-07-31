package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TestSuite configures which tests should be loaded.
type TestSuite struct {
	// The type meta object, should always be a GVK of kudo.dev/v1alpha1/TestSuite.
	metav1.TypeMeta `json:",inline"`
	// Set labels or the test suite name.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Path to CRDs to install before running tests.
	CRDDir string `json:"crdDir"`
	// Paths to directories containing manifests to install before running tests.
	ManifestDirs []string `json:"manifestDirs"`
	// Directories containing test cases to run.
	TestDirs []string `json:"testDirs"`
	// Whether or not to start a local etcd and kubernetes API server for the tests.
	StartControlPlane bool `json:"startControlPlane"`
	// Whether or not to start a local kind cluster for the tests.
	StartKIND bool `json:"startKIND"`
	// Path to the KIND configuration file to use.
	KINDConfig string `json:"kindConfig"`
	// KIND context to use.
	KINDContext string `json:"kindContext"`
	// Whether or not to start the KUDO controller for the tests.
	StartKUDO bool `json:"startKUDO"`
	// If set, do not delete the resources after running the tests (implies SkipClusterDelete).
	SkipDelete bool `json:"skipDelete"`
	// If set, do not delete the mocked control plane or kind cluster.
	SkipClusterDelete bool `json:"skipClusterDelete"`
	// Override the default timeout of 30 seconds (in seconds).
	Timeout int `json:"timeout"`
	// The maximum number of tests to run at once (default: 8).
	Parallel int `json:"parallel"`
	// The directory to output artifacts to (current working directory if not specified).
	ArtifactsDir string `json:"artifactsDir"`
	// Kubectl commands to run before running any tests.
	Kubectl []string `json:"kubectl"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TestStep settings to apply to a test step.
type TestStep struct {
	// The type meta object, should always be a GVK of kudo.dev/v1alpha1/TestStep.
	metav1.TypeMeta `json:",inline"`
	// Override the default metadata. Set labels or override the test step name.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Index int `json:"index,omitempty"`
	// Objects to delete at the beginning of the test step.
	Delete []ObjectReference `json:"delete,omitempty"`

	// Indicates that this is a unit test - safe to run without a real Kubernetes cluster.
	UnitTest bool `json:"unitTest"`

	// Kubectl commands to run at the start of the test
	Kubectl []string `json:"kubectl"`

	// Allowed environment labels
	// Disallowed environment labels
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TestAssert represents the settings needed to verify the result of a test step.
type TestAssert struct {
	// The type meta object, should always be a GVK of kudo.dev/v1alpha1/TestAssert.
	metav1.TypeMeta `json:",inline"`
	// Override the default timeout of 30 seconds (in seconds).
	Timeout int `json:"timeout"`
}

// ObjectReference is a Kubernetes object reference with added labels to allow referencing
// objects by label.
type ObjectReference struct {
	corev1.ObjectReference `json:",inline"`
	// Labels to match on.
	Labels map[string]string
}
