package engine

import (
	"testing"
)

func TestNilTask_Run(t *testing.T) {
	task := &NilTask{}

	if err := task.Run(); err != nil {
		t.Error(err)
	}
}
