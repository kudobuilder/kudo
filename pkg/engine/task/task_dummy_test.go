package task

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDummyTask_Run(t *testing.T) {
	tests := []struct {
		name string
		task DummyTask
		want bool
	}{
		{
			name: "dummy failure",
			task: DummyTask{Fail: true},
			want: false,
		},
		{
			name: "dummy success",
			task: DummyTask{Fail: false},
			want: true,
		},
	}
	ctx := Context{}

	for _, tt := range tests {
		got, _ := tt.task.Run(ctx)
		assert.True(t, got == tt.want, fmt.Sprintf("%s test failed, wanted %t but was %t", tt.name, tt.want, got))
	}
}
