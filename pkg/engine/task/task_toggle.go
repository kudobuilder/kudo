package task

import (
	"fmt"
	"strconv"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
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
	task, err := tt.delegateTask(ctx)
	if err != nil {
		return false, fatalExecutionError(err, toggleTaskError, ctx.Meta)
	}
	// 2. - Run the returned task
	return task.Run(ctx)
}

func (tt ToggleTask) intermediateTaskSpec() kudoapi.TaskSpec {
	return kudoapi.TaskSpec{
		ResourceTaskSpec: kudoapi.ResourceTaskSpec{
			Resources: tt.Resources,
		},
	}
}

func (tt ToggleTask) delegateTask(ctx Context) (Tasker, error) {
	var task Tasker
	// 1. - Get the parameter value
	val, exists := ctx.Parameters[tt.Parameter]
	if !exists {
		return task, fmt.Errorf("no value for parameter %s found", tt.Parameter)
	}

	stringVal, ok := val.(string)
	if !ok {
		return task, fmt.Errorf("value of parameter %s isn't a string", tt.Parameter)
	}

	enabled, err := strconv.ParseBool(stringVal)
	if err != nil {
		return task, fmt.Errorf("could not parse value of parameter %s: %v", tt.Parameter, err)
	}
	// 2. - Return the Apply or Delete task based on parameter value
	if enabled {
		task, err = newApply(&kudoapi.Task{
			Name: tt.Name,
			Kind: ApplyTaskKind,
			Spec: tt.intermediateTaskSpec(),
		})
	} else {
		task, err = newDelete(&kudoapi.Task{
			Name: tt.Name,
			Kind: DeleteTaskKind,
			Spec: tt.intermediateTaskSpec(),
		})
	}
	return task, err
}
