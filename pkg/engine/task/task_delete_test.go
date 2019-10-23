package task

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDeleteTask_Run(t *testing.T) {
	meta := Metadata{
		Metadata: engine.Metadata{
			InstanceName:        "test",
			InstanceNamespace:   "default",
			OperatorName:        "first-operator",
			OperatorVersionName: "first-operator-1.0",
			OperatorVersion:     "1.0",
		},
		PlanName:  "plan",
		PhaseName: "phase",
		StepName:  "step",
		TaskName:  "task",
	}

	tests := []struct {
		name    string
		task    DeleteTask
		done    bool
		wantErr bool
		fatal   bool
		ctx     Context
	}{
		{
			name: "succeeds when there is nothing to delete",
			task: DeleteTask{
				Name:      "install",
				Resources: []string{},
			},
			done:    true,
			wantErr: false,
			ctx: Context{
				Client:   fake.NewFakeClientWithScheme(scheme.Scheme),
				Enhancer: &testKubernetesObjectEnhancer{},
				Meta:     Metadata{},
			},
		},
		{
			name: "fails when a rendering error occurs",
			task: DeleteTask{
				Name:      "task",
				Resources: []string{"pod"},
			},
			done:    false,
			wantErr: true,
			fatal:   true,
			ctx: Context{
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Enhancer:  &testKubernetesObjectEnhancer{},
				Meta:      meta,
				Templates: map[string]string{},
			},
		},
		{
			name: "fails when a kustomizing error occurs",
			task: DeleteTask{
				Name:      "task",
				Resources: []string{"pod"},
			},
			done:    false,
			wantErr: true,
			fatal:   true,
			ctx: Context{
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Enhancer:  &errKubernetesObjectEnhancer{},
				Meta:      meta,
				Templates: map[string]string{"pod": resourceAsString(pod("pod1", "default"))},
			},
		},
		{
			name: "succeeds to delete a resource",
			task: DeleteTask{
				Name:      "task",
				Resources: []string{"pod"},
			},
			done:    true,
			wantErr: false,
			ctx: Context{
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Enhancer:  &testKubernetesObjectEnhancer{},
				Meta:      meta,
				Templates: map[string]string{"pod": resourceAsString(pod("pod1", "default"))},
			},
		},
	}

	for _, tt := range tests {
		got, err := tt.task.Run(tt.ctx)
		assert.True(t, tt.done == got, fmt.Sprintf("%s failed: want = %t, wantErr = %v", tt.name, got, err))
		if tt.wantErr {
			assert.True(t, errors.Is(err, ErrFatalExecution) == tt.fatal, "expected a fatal: %t error", tt.fatal)
			assert.Error(t, err)
		}
		if !tt.wantErr {
			assert.NoError(t, err)
		}
	}
}
