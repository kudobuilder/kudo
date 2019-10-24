package engine

import (
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

	// the object that will own all the resources created by this execution
	ResourcesOwner metav1.Object
}

// ExecutionError wraps plan execution engine errors with additional fields. E.g an error with EventName set
// will be published on the event bus.
type ExecutionError struct {
	Err       error
	Fatal     bool    // these errors should not be retried
	EventName *string // nil if no warn even should be created
}

func (e ExecutionError) Error() string {
	if e.Fatal {
		return fmt.Sprintf("Fatal error: %v", e.Err)
	}
	return fmt.Sprintf("Error during execution: %v", e.Err)
}
