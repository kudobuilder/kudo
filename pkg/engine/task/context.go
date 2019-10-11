package task

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ExecutionMetadata contains ExecutionMetadata associated with current plan being executed
type ExecutionMetadata struct {
	EngineMetadata

	PlanName  string
	PhaseName string
	StepName  string
	TaskName  string
}

type EngineMetadata struct {
	InstanceName        string
	InstanceNamespace   string
	OperatorName        string
	OperatorVersionName string
	OperatorVersion     string

	// the object that will own all the resources created by this execution
	ResourcesOwner metav1.Object
}

type Context struct {
	Client     client.Client
	Enhancer   KubernetesObjectEnhancer
	Meta       ExecutionMetadata
	Templates  map[string]string // Raw templates
	Parameters map[string]string // I and OV parameters merged
}
