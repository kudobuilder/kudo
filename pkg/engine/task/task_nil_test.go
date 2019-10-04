package task

import (
	"testing"
)

func TestNilTask_Run(t *testing.T) {
	task := &NilTask{}
	ctx := Context{}

	if err := task.Run(ctx); err != nil {
		t.Error(err)
	}
}
