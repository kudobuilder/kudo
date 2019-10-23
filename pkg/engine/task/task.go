package task

import (
	"errors"
	"fmt"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/engine"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Metadata contains Metadata along with specific fields associated with current plan
// being executed like current plan, phase, step or task names.
type Metadata struct {
	engine.Metadata

	PlanName  string
	PhaseName string
	StepName  string
	TaskName  string
}

// Context is a engine.task execution context containing k8s client, templates parameters etc.
type Context struct {
	Client     client.Client
	Enhancer   Enhancer
	Meta       Metadata
	Templates  map[string]string // Raw templates
	Parameters map[string]string // Instance and OperatorVersion parameters merged
}

// Tasker is an interface that represents any runnable task for an operator. This method is treated
// as idempotent and will be called multiple times during the life-cycle of the plan execution.
// Method returns a boolean, signalizing that the task has finished successfully, and an error.
// An error can wrap the ErrFatalExecution for errors that should not be retried e.g. failed template
// rendering. This will result in a v1alpha1.ExecutionFatalError in the plan execution status. A normal
// error e.g. failure during accessing the API server will be treated  as "transient", meaning plan
// execution engine  can retry it next time. Only a (true, nil) return value will be treated as successful
// task execution.
type Tasker interface {
	Run(ctx Context) (bool, error)
}

// Available tasks kinds
const (
	ApplyTaskKind  = "Apply"
	DeleteTaskKind = "Delete"
	DummyTaskKind  = "Dummy"
)

var (
	// ErrFatalExecution is a wrapper for the fatal engine task execution error
	ErrFatalExecution = errors.New("fatal task error: ")
)

// Build factory method takes an v1alpha1.Task and returns a corresponding Tasker object
func Build(task *v1alpha1.Task) (Tasker, error) {
	switch task.Kind {
	case ApplyTaskKind:
		return newApply(task), nil
	case DeleteTaskKind:
		return newDelete(task), nil
	case DummyTaskKind:
		return newDummy(task), nil
	default:
		return nil, fmt.Errorf("%wunknown task kind %s", ErrFatalExecution, task.Kind)
	}
}

func newApply(task *v1alpha1.Task) ApplyTask {
	return ApplyTask{
		Name:      task.Name,
		Resources: task.Spec.ResourceTaskSpec.Resources,
	}
}

func newDelete(task *v1alpha1.Task) DeleteTask {
	return DeleteTask{
		Name:      task.Name,
		Resources: task.Spec.ResourceTaskSpec.Resources,
	}
}

func newDummy(task *v1alpha1.Task) DummyTask {
	return DummyTask{
		Name:    task.Name,
		WantErr: task.Spec.DummyTaskSpec.WantErr,
		Fatal:   task.Spec.DummyTaskSpec.Fatal,
		Done:    task.Spec.DummyTaskSpec.Done,
	}
}
