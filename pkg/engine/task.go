package engine

import (
	"encoding/json"
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
)

var (
	KCreateTask = "create"
	KNullTask   = "null"
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
	Name string `json:"name"`
	Kind string `json:"kind"`
	NullTask
	CreateTask
}

// UnmarshalJSON is a custom JSON unmarshaler that determines the type of task
func (t *Task) UnmarshalJSON(b []byte) error {
	var objMap map[string]*json.RawMessage
	if err := json.Unmarshal(b, &objMap); err != nil {
		return err
	}

	var name string
	if err := json.Unmarshal(*objMap["name"], &name); err != nil {
		return err
	}
	t.Name = name

	var kind string
	if err := json.Unmarshal(*objMap["kind"], &kind); err != nil {
		return err
	}
	t.Kind = kind

	var spec map[string]interface{}
	if err := json.Unmarshal(*objMap["spec"], &spec); err != nil {
		return err
	}

	switch t.Kind {
	case KCreateTask:
		t.CreateTask = CreateTask{}
	default:
		return ErrNoValidTaskNames
	}

	return nil
}

// Run is the entrypoint function to run a task, polymorphically determining which task to run and run it
func (t *Task) Run() error {
	var task Tasker
	switch t.Kind {
	case KCreateTask:
		task = &CreateTask{
			Resources: []runtime.Object{},
		}
		break
	case KNullTask:
		task = &t.NullTask
		break
	default:
		return ErrNoValidTaskNames
	}

	return task.Run()
}
