package task

import (
	"fmt"
	"testing"

	"github.com/kudobuilder/kudo/pkg/util/template"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

func TestApplyTask_Run(t *testing.T) {
	meta := ExecutionMetadata{
		EngineMetadata: EngineMetadata{
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
				Client:   fake.NewFakeClientWithScheme(scheme.Scheme),
				Enhancer: &testKubernetesObjectEnhancer{},
				Meta:     ExecutionMetadata{},
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
			ctx: Context{
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Enhancer:  &testKubernetesObjectEnhancer{},
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
			ctx: Context{
				Client:    fake.NewFakeClientWithScheme(scheme.Scheme),
				Enhancer:  &errKubernetesObjectEnhancer{},
				Meta:      meta,
				Templates: map[string]string{"pod": resourceAsString(pod("pod1", "default"))},
			},
		},
		{
			name: "succeeds when the resource is healthy (unknown type Pod is marked healthy by default)",
			task: ApplyTask{
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
				Enhancer:  &testKubernetesObjectEnhancer{},
				Meta:      meta,
				Templates: map[string]string{"job": resourceAsString(job("job1", "default"))},
			},
		},
	}

	for _, tt := range tests {
		got, err := tt.task.Run(tt.ctx)
		assert.True(t, tt.done == got, fmt.Sprintf("%s failed: want = %t, wantErr = %v", tt.name, got, err))
		if tt.wantErr {
			assert.Error(t, err)
		}
		if !tt.wantErr {
			assert.NoError(t, err)
		}
	}
}

func pod(name string, namespace string) *corev1.Pod {
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

type testKubernetesObjectEnhancer struct{}

func (k *testKubernetesObjectEnhancer) ApplyConventionsToTemplates(templates map[string]string, metadata ExecutionMetadata) ([]runtime.Object, error) {
	result := make([]runtime.Object, 0)
	for _, t := range templates {
		objsToAdd, err := template.ParseKubernetesObjects(t)
		if err != nil {
			return nil, errors.Wrapf(err, "error parsing kubernetes objects after applying kustomize")
		}
		result = append(result, objsToAdd[0])
	}
	return result, nil
}

type errKubernetesObjectEnhancer struct{}

func (k *errKubernetesObjectEnhancer) ApplyConventionsToTemplates(templates map[string]string, metadata ExecutionMetadata) ([]runtime.Object, error) {
	return nil, errors.New("always error")
}
