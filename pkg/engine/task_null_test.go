package engine

import (
	"testing"
)

func TestNullTask_Run(t *testing.T) {
	task := &NullTask{}

	if err := task.Run(); err != nil {
		t.Error(err)
	}
}
