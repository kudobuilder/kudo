package task

import (
	"errors"
	"fmt"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
)

// Tasker is an interface that represents any runnable task for an operator
type Tasker interface {
	Run(ctx Context) (bool, error)
}

const (
	ApplyTaskKind  = "Apply"
	DeleteTaskKind = "Delete"
	NilTaskKind    = "Nil"
)

var (
	ErrInvalidTaskKind = errors.New("invalid task kind")
)

// Builder factory method takes an v1alpha1.Task and returns a corresponding Tasker object
func Builder(task v1alpha1.Task) (Tasker, error) {
	switch task.Kind {
	case ApplyTaskKind:
		return newApply(task), nil
	case DeleteTaskKind:
		return newDelete(task), nil
	case NilTaskKind:
		return newNil(task), nil
	default:
		return nil, fmt.Errorf("unknown task kind %s: %w", task.Kind, ErrInvalidTaskKind)
	}
}

func newApply(task v1alpha1.Task) ApplyTask {
	return ApplyTask{
		Name:      task.Name,
		Resources: task.Spec.ApplyTaskSpec.Resources,
	}
}

func newDelete(task v1alpha1.Task) DeleteTask {
	return DeleteTask{
		Name:      task.Name,
		Resources: task.Spec.DeleteTaskSpec.Resources,
	}
}

func newNil(task v1alpha1.Task) NilTask {
	return NilTask{}
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
