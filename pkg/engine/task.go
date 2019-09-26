package engine

import (
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
)

const (
	ApplyTaskKind  = "Apply"
	DeleteTaskKind = "Delete"
	NilTaskKind    = "Nil"
)

var (
	ErrNoValidTaskNames = errors.New("no valid task names found")
)

// Tasker is an interface that represents any runnable task for an operator
type Tasker interface {
	Run() error
}

type Renderer interface {
	Render() interface{}
}

// Task is a global, polymorphic implementation of all publicly available tasks
// +k8s:deepcopy-gen=true
type Task struct {
	Name string   `json:"name"`
	Kind string   `json:"kind"`
	Spec TaskSpec `json:"spec"`
}

// +k8s:deepcopy-gen=true
type TaskSpec struct {
	NilTask
	ApplyTask
	DeleteTask
}

// Run is the entrypoint function to run a task, polymorphically determining which task to run and run it
func (t *Task) Run() error {
	var task Tasker
	switch t.Kind {
	case ApplyTaskKind:
		task = &ApplyTask{
			Resources: []runtime.Object{},
		}
		break
	case NilTaskKind:
		task = &NilTask{}
		break
	case DeleteTaskKind:
		task = &DeleteTask{
			Resources: []runtime.Object{},
		}
	default:
		return ErrNoValidTaskNames
	}

	return task.Run()
}
