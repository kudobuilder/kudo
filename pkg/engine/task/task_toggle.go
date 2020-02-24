package task

import (
	"fmt"
	"strconv"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
)

const (
	toggleTaskError = "ToggleTaskError"
)

// ToggleTask will apply or delete a set of given resources to the cluster based on value of Parameter. See Run method for more details.
type ToggleTask struct {
	Name      string
	Parameter string
	Resources []string
}

func (tt ToggleTask) Run(ctx Context) (bool, error) {
	// 1. - Get the task to run
	task, err := tt.getTask(ctx)
	if err != nil {
		return false, fatalExecutionError(err, toggleTaskError, ctx.Meta)
	}
	// 2. - Run the returned task
	return task.Run(ctx)
}

func (tt ToggleTask) convertToTaskSpec() v1beta1.TaskSpec {
	return v1beta1.TaskSpec{
		ResourceTaskSpec: v1beta1.ResourceTaskSpec{
			Resources: tt.Resources,
		},
	}
}

func (tt ToggleTask) getTask(ctx Context) (Tasker, error) {
	var task Tasker
	// 1. - Get the parameter value
	paramValue := ctx.Parameters[tt.Parameter]
	if len(paramValue) == 0 {
		return task, fmt.Errorf("empty value for parameter %s", tt.Parameter)
	}
	enabled, err := strconv.ParseBool(paramValue)
	if err != nil {
		return task, err
	}
	// 2. - Return the Apply or Delete task based on parameter value
	if enabled {
		task, err = newApply(&v1beta1.Task{
			Name: tt.Name,
			Kind: ApplyTaskKind,
			Spec: tt.convertToTaskSpec(),
		})
	} else {
		task, err = newDelete(&v1beta1.Task{
			Name: tt.Name,
			Kind: DeleteTaskKind,
			Spec: tt.convertToTaskSpec(),
		})

	}
	return task, err
}
