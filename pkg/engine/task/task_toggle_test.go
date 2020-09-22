package task

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/engine/renderer"
	kudofake "github.com/kudobuilder/kudo/pkg/test/fake"
)

func TestToggleTask_Run(t *testing.T) {
	meta := renderer.Metadata{
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
		task    ToggleTask
		done    bool
		wantErr bool
		fatal   bool
		ctx     Context
	}{
		{
			name: "fails when no parameter is passed",
			task: ToggleTask{
				Name:      "task",
				Resources: []string{"pod"},
				Parameter: "feature-enabled",
			},
			done:    false,
			wantErr: true,
			fatal:   true,
			ctx: Context{
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Discovery: kudofake.CachedDiscoveryClient(),
				Enhancer:  &testEnhancer{},
				Meta:      meta,
				Templates: map[string]string{},
			},
		},
		{
			name: "succeeds when the resource is healthy",
			task: ToggleTask{
				Name:      "task",
				Resources: []string{"pod"},
				Parameter: "feature-enabled",
			},
			done:    true,
			wantErr: false,
			ctx: Context{
				Parameters: map[string]interface{}{
					"feature-enabled": "true",
				},
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Discovery: kudofake.CachedDiscoveryClient(),
				Enhancer:  &testEnhancer{},
				Meta:      meta,
				Templates: map[string]string{"pod": resourceAsString(pod("pod1", "default"))},
			},
		},
		{
			name: "is not done when the resource is unhealthy",
			task: ToggleTask{
				Name:      "task",
				Resources: []string{"job"},
				Parameter: "feature-enabled",
			},
			done:    false,
			wantErr: false,
			ctx: Context{
				Parameters: map[string]interface{}{
					"feature-enabled": "true",
				},
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Discovery: kudofake.CachedDiscoveryClient(),
				Enhancer:  &testEnhancer{},
				Meta:      meta,
				Templates: map[string]string{"job": resourceAsString(job("job1", "default"))},
			},
		},
	}

	for _, tt := range tests {
		got, err := tt.task.Run(tt.ctx)
		assert.True(t, tt.done == got, fmt.Sprintf("%s failed: want = %t, wantErr = %v", tt.name, got, err))
		if tt.wantErr {
			assert.True(t, errors.Is(err, engine.ErrFatalExecution) == tt.fatal)
			assert.Error(t, err)
		}
		if !tt.wantErr {
			assert.NoError(t, err)
		}
	}
}

func TestToggleTask_intermediateTask(t *testing.T) {
	meta := renderer.Metadata{
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
		name             string
		task             ToggleTask
		wantErr          bool
		ctx              Context
		expectedTaskType reflect.Type
	}{
		{
			name: "use apply task when parameter is true",
			task: ToggleTask{
				Name:      "task",
				Resources: []string{"pod"},
				Parameter: "feature-enabled",
			},
			wantErr: false,
			ctx: Context{
				Parameters: map[string]interface{}{
					"feature-enabled": "true",
				},
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Discovery: kudofake.CachedDiscoveryClient(),
				Enhancer:  &testEnhancer{},
				Meta:      renderer.Metadata{},
				Templates: map[string]string{"pod": resourceAsString(pod("pod1", "default"))},
			},
			expectedTaskType: reflect.TypeOf(ApplyTask{}),
		},
		{
			name: "use delete task when parameter is false",
			task: ToggleTask{
				Name:      "task",
				Resources: []string{"pod"},
				Parameter: "feature-enabled",
			},
			wantErr: false,
			ctx: Context{
				Parameters: map[string]interface{}{
					"feature-enabled": "false",
				},
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Discovery: kudofake.CachedDiscoveryClient(),
				Enhancer:  &testEnhancer{},
				Meta:      renderer.Metadata{},
				Templates: map[string]string{"pod": resourceAsString(pod("pod1", "default"))},
			},
			expectedTaskType: reflect.TypeOf(DeleteTask{}),
		},
		{
			name: "fails when parameter is not a boolean",
			task: ToggleTask{
				Name:      "task",
				Resources: []string{"pod"},
				Parameter: "feature-enabled",
			},
			wantErr: true,
			ctx: Context{
				Parameters: map[string]interface{}{
					"feature-enabled": "notABooleanValue",
				},
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Discovery: kudofake.CachedDiscoveryClient(),
				Enhancer:  &testEnhancer{},
				Meta:      meta,
				Templates: map[string]string{},
			},
			expectedTaskType: reflect.TypeOf(nil),
		},
		{
			name: "fails when parameter is empty",
			task: ToggleTask{
				Name:      "task",
				Resources: []string{"pod"},
				Parameter: "feature-enabled",
			},
			wantErr: true,
			ctx: Context{
				Parameters: map[string]interface{}{
					"feature-enabled": "",
				},
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Discovery: kudofake.CachedDiscoveryClient(),
				Enhancer:  &testEnhancer{},
				Meta:      meta,
				Templates: map[string]string{},
			},
			expectedTaskType: reflect.TypeOf(nil),
		},
		{
			name: "fails when parameter is not present",
			task: ToggleTask{
				Name:      "task",
				Resources: []string{"pod"},
			},
			wantErr: true,
			ctx: Context{
				Parameters: map[string]interface{}{
					"feature-enabled": "someValue",
				},
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Discovery: kudofake.CachedDiscoveryClient(),
				Enhancer:  &testEnhancer{},
				Meta:      meta,
				Templates: map[string]string{},
			},
			expectedTaskType: reflect.TypeOf(nil),
		},
	}

	for _, tt := range tests {
		got, err := tt.task.delegateTask(tt.ctx)
		assert.True(t, tt.expectedTaskType == reflect.TypeOf(got), fmt.Sprintf("%s failed: want = %t, wantErr = %v", tt.name, got, err))

		if tt.wantErr {
			assert.Error(t, err)
		}
		if !tt.wantErr {
			assert.NoError(t, err)
		}
	}
}
