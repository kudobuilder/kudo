package task

import (
	"errors"
	"fmt"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
)

// Tasker is an interface that represents any runnable task for an operator. This method is treated
// as idempotent and will be called multiple times during the life-cycle of the plan execution.
// Method returns a boolean, signalizing that the task has finished successfully, and an error.
// An error can wrap the FatalExecutionError for errors that can not be retried e.g. failed template
// rendering. This will result in a v1alpha1.ExecutionFatalError in the plan execution status. A normal
// error e.g. failure during accessing the API server will be treated  as "transient", meaning plan
// execution engine  can retry it next time. Only a (true, nil) return value will be treated as successful
// task execution.
type Tasker interface {
	Run(ctx Context) (bool, error)
}

const (
	ApplyTaskKind  = "Apply"
	DeleteTaskKind = "Delete"
	DummyTaskKind  = "Dummy"
)

var (
	FatalExecutionError = errors.New("")
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
		return nil, fmt.Errorf("unknown task kind %s", task.Kind)
	}
}

func newApply(task *v1alpha1.Task) ApplyTask {
	return ApplyTask{
		Name:      task.Name,
		Resources: task.Spec.ApplyTaskSpec.Resources,
	}
}

func newDelete(task *v1alpha1.Task) DeleteTask {
	return DeleteTask{
		Name:      task.Name,
		Resources: task.Spec.DeleteTaskSpec.Resources,
	}
}

func newDummy(task *v1alpha1.Task) DummyTask {
	return DummyTask{Fail: task.Spec.DummyTaskSpec.Fail}
}

// An example of new TaskSpec
//tasks:
//	- name: helmExample
//	  kind: Helm
//	  spec:
//		baseChart: //some/helm/url
//		...
//	- name: applyExample
//	  kind: Apply
//	  spec:
//	  	applyResources:
//	  	  - pdb.yaml
//		  - deployment.yaml
//	- name: deleteExample
//	  kind: Delete
//  	  spec:
//		deleteResources:
//			- pod.yaml
//			- service.yaml
//	- namme: pipeExample
//	  kind: Pipe
//	  spec:
//		containerSpec:
//			...
//		pipe:
//			file: /usr/share/MyKey.key
//			kind: Secret # ConfigMap
//			key: {{.Pipes.Certificate}}
