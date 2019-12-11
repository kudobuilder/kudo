package task

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/engine"
)

func TestDummyTask_Run(t *testing.T) {
	tests := []struct {
		name string
		task DummyTask
		want bool
	}{
		{
			name: "dummy error",
			task: DummyTask{WantErr: true},
			want: false,
		},
		{
			name: "fatal dummy error",
			task: DummyTask{WantErr: true, Fatal: true},
			want: false,
		},
		{
			name: "dummy not done",
			task: DummyTask{WantErr: false},
			want: false,
		},
		{
			name: "dummy done",
			task: DummyTask{Done: true},
			want: true,
		},
	}
	ctx := Context{}

	for _, tt := range tests {
		got, err := tt.task.Run(ctx)
		assert.True(t, got == tt.want, fmt.Sprintf("%s test failed, wanted %t but was %t", tt.name, tt.want, got))

		if tt.task.Fatal {
			assert.True(t, errors.Is(err, engine.ErrFatalExecution))
		}
	}
}
