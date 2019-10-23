package engine

import (
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
