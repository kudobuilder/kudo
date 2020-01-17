package engine

import (
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Metadata contains engine metadata associated with the current operator being executed
type Metadata struct {
	InstanceName        string
	InstanceNamespace   string
	OperatorName        string
	OperatorVersionName string
	OperatorVersion     string
	AppVersion          string

	// the object that will own all the resources created by this execution
	ResourcesOwner metav1.Object
}

var (
	// ErrFatalExecution is a wrapper for the fatal engine task execution error
	ErrFatalExecution = errors.New("fatal error: ")
)

// ExecutionError wraps plan execution engine errors with additional fields. An execution error will be published
// on the event bus using provide EventName as a reason.
type ExecutionError struct {
	Err       error
	EventName string
}

func (e ExecutionError) Error() string {
	return fmt.Sprintf("Error during execution: %v", e.Err)
}

func (e ExecutionError) Unwrap() error { return e.Err }
