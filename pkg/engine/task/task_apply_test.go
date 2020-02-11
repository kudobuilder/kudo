package task

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/engine/renderer"
	"github.com/kudobuilder/kudo/pkg/test/utils"
)

func TestApplyTask_Run(t *testing.T) {
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
		task    ApplyTask
		done    bool
		wantErr bool
		fatal   bool
		ctx     Context
	}{
		{
			name: "succeeds when there is nothing to apply",
			task: ApplyTask{
				Name:      "install",
				Resources: []string{},
			},
			done:    true,
			wantErr: false,
			ctx: Context{
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Discovery: utils.FakeDiscoveryClient(),
				Enhancer:  &testEnhancer{},
				Meta:      renderer.Metadata{},
			},
		},
		{
			name: "fails when a rendering error occurs",
			task: ApplyTask{
				Name:      "task",
				Resources: []string{"pod"},
			},
			done:    false,
			wantErr: true,
			fatal:   true,
			ctx: Context{
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Discovery: utils.FakeDiscoveryClient(),
				Enhancer:  &testEnhancer{},
				Meta:      meta,
				Templates: map[string]string{},
			},
		},
		{
			name: "fails when a kustomizing error occurs",
			task: ApplyTask{
				Name:      "task",
				Resources: []string{"pod"},
			},
			done:    false,
			wantErr: true,
			fatal:   true,
			ctx: Context{
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Discovery: utils.FakeDiscoveryClient(),
				Enhancer:  &errorEnhancer{},
				Meta:      meta,
				Templates: map[string]string{"pod": resourceAsString(pod("pod1", "default"))},
			},
		},
		{
			name: "succeeds when the resource is healthy (pod has PodStatus.Phase = Running)",
			task: ApplyTask{
				Name:      "task",
				Resources: []string{"pod"},
			},
			done:    true,
			wantErr: false,
			ctx: Context{
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Discovery: utils.FakeDiscoveryClient(),
				Enhancer:  &testEnhancer{},
				Meta:      meta,
				Templates: map[string]string{"pod": resourceAsString(pod("pod1", "default"))},
			},
		},
		{
			name: "is not done when the resource is unhealthy",
			task: ApplyTask{
				Name:      "task",
				Resources: []string{"job"},
			},
			done:    false,
			wantErr: false,
			ctx: Context{
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Discovery: utils.FakeDiscoveryClient(),
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

func pod(name string, namespace string) *corev1.Pod { //nolint:unparam
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}
	return pod
}

func job(name string, namespace string) *batchv1.Job {
	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Job",
			APIVersion: "batch/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{},
	}
	return job
}

func resourceAsString(resource metav1.Object) string {
	bytes, _ := yaml.Marshal(resource)
	return string(bytes)
}

type testEnhancer struct{}

func (k *testEnhancer) Apply(templates map[string]string, metadata renderer.Metadata) ([]runtime.Object, error) {
	result := make([]runtime.Object, 0)
	for _, t := range templates {
		objsToAdd, err := renderer.YamlToObject(t)
		if err != nil {
			return nil, fmt.Errorf("error parsing kubernetes objects after applying enhance: %w", err)
		}
		result = append(result, objsToAdd[0])
	}
	return result, nil
}

type errorEnhancer struct{}

func (k *errorEnhancer) Apply(templates map[string]string, metadata renderer.Metadata) ([]runtime.Object, error) {
	return nil, errors.New("always error")
}
