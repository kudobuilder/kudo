package task

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/engine/renderer"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
)

// Context is a engine.task execution context containing k8s client, templates parameters etc.
type Context struct {
	Client     client.Client
	Enhancer   renderer.Enhancer
	Meta       renderer.Metadata
	Templates  map[string]string // Raw templates
	Parameters map[string]string // Instance and OperatorVersion parameters merged
}

// Tasker is an interface that represents any runnable task for an operator. This method is treated
// as idempotent and will be called multiple times during the life-cycle of the plan execution.
// Method returns a boolean, signalizing that the task has finished successfully, and an error.
// An error can wrap the ErrFatalExecution for errors that should not be retried e.g. failed template
// rendering. This will result in a v1beta1.ExecutionFatalError in the plan execution status. A normal
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
	taskRenderingError   = "TaskRenderingError"
	taskEnhancementError = "TaskEnhancementError"
	dummyTaskError       = "DummyTaskError"
)

// Build factory method takes an v1beta1.Task and returns a corresponding Tasker object
func Build(task *v1beta1.Task) (Tasker, error) {
	switch task.Kind {
	case ApplyTaskKind:
		return newApply(task), nil
	case DeleteTaskKind:
		return newDelete(task), nil
	case DummyTaskKind:
		return newDummy(task), nil
	default:
		return nil, fmt.Errorf("unknown task kind %s", task.Kind)
	}
}

func newApply(task *v1beta1.Task) ApplyTask {
	return ApplyTask{
		Name:      task.Name,
		Resources: task.Spec.ResourceTaskSpec.Resources,
	}
}

func newDelete(task *v1beta1.Task) DeleteTask {
	return DeleteTask{
		Name:      task.Name,
		Resources: task.Spec.ResourceTaskSpec.Resources,
	}
}

func newDummy(task *v1beta1.Task) DummyTask {
	return DummyTask{
		Name:    task.Name,
		WantErr: task.Spec.DummyTaskSpec.WantErr,
		Fatal:   task.Spec.DummyTaskSpec.Fatal,
		Done:    task.Spec.DummyTaskSpec.Done,
	}
}

// fatalExecutionError is a helper method providing proper wrapping an message for the ExecutionError
func fatalExecutionError(cause error, eventName string, meta renderer.Metadata) engine.ExecutionError {
	return engine.ExecutionError{
		Err: fmt.Errorf("%w%s/%s failed in %s.%s.%s.%s: %v",
			engine.ErrFatalExecution,
			meta.InstanceNamespace,
			meta.InstanceName,
			meta.PlanName,
			meta.PhaseName,
			meta.StepName,
			meta.TaskName,
			cause),
		EventName: eventName,
	}
}
