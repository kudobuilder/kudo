package task

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ExecutionMetadata contains ExecutionMetadata along with specific fields associated with current plan
// being executed like current plan, phase, step or task names.
type ExecutionMetadata struct {
	EngineMetadata

	PlanName  string
	PhaseName string
	StepName  string
	TaskName  string
}

// EngineMetadata contains metadata associated with the current operator being executed
type EngineMetadata struct {
	InstanceName        string
	InstanceNamespace   string
	OperatorName        string
	OperatorVersionName string
	OperatorVersion     string

	// the object that will own all the resources created by this execution
	ResourcesOwner metav1.Object
}

// Context is a engine.task execution context containing k8s client, templates parameters etc.
type Context struct {
	Client     client.Client
	Enhancer   KubernetesObjectEnhancer
	Meta       ExecutionMetadata
	Templates  map[string]string // Raw templates
	Parameters map[string]string // Instance and OperatorVersion parameters merged
}
