package task

import (
	"errors"
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
	Run(ctx Context) error
}

type Outputter interface {
	Output() interface{}
}

type TaskBuilder func(input interface{}) (Tasker, error)

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
func (t *Task) Run(ctx Context) error {
	var task Tasker
	switch t.Kind {
	case ApplyTaskKind:
		task = &ApplyTask{
			Resources: []string{},
		}
		break
	case NilTaskKind:
		task = &NilTask{}
		break
	//case DeleteTaskKind:
	//	task = &DeleteTask{
	//		Resources: []runtime.Object{},
	//	}
	default:
		return ErrNoValidTaskNames
	}

	return task.Run(ctx)
}

//------
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
