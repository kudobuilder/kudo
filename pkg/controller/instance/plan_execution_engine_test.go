package instance

import (
	"reflect"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/engine/task"
	engtask "github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/util/template"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestExecutePlan(t *testing.T) {
	instance := instance()
	meta := &engtask.EngineMetadata{
		InstanceName:        instance.Name,
		InstanceNamespace:   instance.Namespace,
		OperatorName:        "first-operator",
		OperatorVersionName: "first-operator-1.0",
		OperatorVersion:     "1.0",
		ResourcesOwner:      instance,
	}
	enhancer := &testKubernetesObjectEnhancer{}
	errEnhancer := &errKubernetesObjectEnhancer{}

	tests := []struct {
		name           string
		activePlan     *activePlan
		metadata       *task.EngineMetadata
		expectedStatus *v1alpha1.PlanStatus
		wantErr        bool
		kustomizer     task.KubernetesObjectEnhancer
	}{
		{name: "plan already finished will not change its status", activePlan: &activePlan{
			name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionComplete,
			},
		},
			metadata:       meta,
			expectedStatus: &v1alpha1.PlanStatus{Status: v1alpha1.ExecutionComplete},
			kustomizer:     enhancer,
		},
		{name: "plan with one step to be executed, still in progress", activePlan: &activePlan{
			name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionPending,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionPending, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionPending, Name: "step"}}}},
			},
			spec: &v1alpha1.Plan{
				Strategy: "serial",
				Phases: []v1alpha1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1alpha1.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			tasks: []v1alpha1.Task{
				{
					Name: "task",
					Kind: "Apply",
					Spec: v1alpha1.TaskSpec{
						ApplyTaskSpec: v1alpha1.ApplyTaskSpec{Resources: []string{"job"}},
					},
				},
			},
			templates: map[string]string{"job": getResourceAsString(job("job1", "default"))},
		},
			metadata: meta,
			expectedStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionInProgress, Name: "step"}}}}},
			kustomizer: enhancer},
		// this plan deploys pod, that is marked as healthy immediately because we cannot evaluate health
		{name: "plan with one step, immediately healthy -> completed", activePlan: &activePlan{
			name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionPending,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionPending, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionPending, Name: "step"}}}},
			},
			spec: &v1alpha1.Plan{
				Strategy: "serial",
				Phases: []v1alpha1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1alpha1.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			tasks: []v1alpha1.Task{
				{
					Name: "task",
					Kind: "Apply",
					Spec: v1alpha1.TaskSpec{
						ApplyTaskSpec: v1alpha1.ApplyTaskSpec{Resources: []string{"pod"}},
					},
				},
			},
			templates: map[string]string{"pod": getResourceAsString(pod("pod1", "default"))},
		},
			metadata: meta,
			expectedStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionComplete,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionComplete, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionComplete, Name: "step"}}}},
			},
			kustomizer: enhancer,
		},
		{name: "plan in errored state will be retried and completed when no error happens", activePlan: &activePlan{
			name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ErrorStatus,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ErrorStatus, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ErrorStatus, Name: "step"}}}},
			},
			spec: &v1alpha1.Plan{
				Strategy: "serial",
				Phases: []v1alpha1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1alpha1.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			tasks: []v1alpha1.Task{
				{
					Name: "task",
					Kind: "Apply",
					Spec: v1alpha1.TaskSpec{
						ApplyTaskSpec: v1alpha1.ApplyTaskSpec{Resources: []string{"pod"}},
					},
				},
			},
			templates: map[string]string{"pod": getResourceAsString(pod("pod1", "default"))},
		},
			metadata: meta,
			expectedStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionComplete,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionComplete, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionComplete, Name: "step"}}}},
			},
			kustomizer: enhancer,
		},
		// --- Proper error and fatal error status propagation ---
		{name: "plan in progress, will have step error status, when a task fails", activePlan: &activePlan{
			name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Name: "step", Status: v1alpha1.ExecutionInProgress}}}},
			},
			spec: &v1alpha1.Plan{
				Strategy: "serial",
				Phases: []v1alpha1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1alpha1.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			tasks: []v1alpha1.Task{
				{
					Name: "task",
					Kind: "Dummy",
					Spec: v1alpha1.TaskSpec{
						DummyTaskSpec: v1alpha1.DummyTaskSpec{WantErr: true},
					},
				},
			},
			templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ErrorStatus, Name: "step"}}}},
			},
			kustomizer: enhancer,
		},
		{name: "plan in progress, will have plan/phase/step fatal error status, when a task fails with a fatal error", activePlan: &activePlan{
			name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Name: "step", Status: v1alpha1.ExecutionInProgress}}}},
			},
			spec: &v1alpha1.Plan{
				Strategy: "serial",
				Phases: []v1alpha1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1alpha1.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			tasks: []v1alpha1.Task{
				{
					Name: "task",
					Kind: "Dummy",
					Spec: v1alpha1.TaskSpec{
						DummyTaskSpec: v1alpha1.DummyTaskSpec{WantErr: true, Fatal: true},
					},
				},
			},
			templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionFatalError,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionFatalError, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionFatalError, Name: "step"}}}},
			},
			wantErr:    true,
			kustomizer: enhancer,
		},
		{name: "plan in progress with a misconfigured task will fail with a fatal error", activePlan: &activePlan{
			name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Name: "step", Status: v1alpha1.ExecutionInProgress}}}},
			},
			spec: &v1alpha1.Plan{
				Strategy: "serial",
				Phases: []v1alpha1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1alpha1.Step{{Name: "step", Tasks: []string{"fake-task"}}}},
				},
			},
			tasks: []v1alpha1.Task{
				{
					Name: "task",
					Kind: "Dummy",
					Spec: v1alpha1.TaskSpec{
						DummyTaskSpec: v1alpha1.DummyTaskSpec{WantErr: false},
					},
				},
			},
			templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionFatalError,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionFatalError, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionFatalError, Name: "step"}}}},
			},
			wantErr:    true,
			kustomizer: enhancer,
		},
		{name: "plan in progress with an unknown task spec will fail with a fatal error", activePlan: &activePlan{
			name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Name: "step", Status: v1alpha1.ExecutionInProgress}}}},
			},
			spec: &v1alpha1.Plan{
				Strategy: "serial",
				Phases: []v1alpha1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1alpha1.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			tasks: []v1alpha1.Task{
				{
					Name: "task",
					Kind: "Unknown",
					Spec: v1alpha1.TaskSpec{},
				},
			},
			templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionFatalError,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionFatalError, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionFatalError, Name: "step"}}}},
			},
			wantErr:    true,
			kustomizer: enhancer,
		},
		// --- Respect the Steps execution strategy ---
		{name: "plan in progress with multiple serial steps, will respect serial step strategy and stop after first step fails", activePlan: &activePlan{
			name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{
					{Name: "stepOne", Status: v1alpha1.ExecutionInProgress},
					{Name: "stepTwo", Status: v1alpha1.ExecutionInProgress},
				}}},
			},
			spec: &v1alpha1.Plan{
				Strategy: "serial",
				Phases: []v1alpha1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1alpha1.Step{
						{Name: "stepOne", Tasks: []string{"taskOne"}},
						{Name: "stepTwo", Tasks: []string{"taskTwo"}},
					}},
				},
			},
			tasks: []v1alpha1.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: v1alpha1.TaskSpec{
						DummyTaskSpec: v1alpha1.DummyTaskSpec{WantErr: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: v1alpha1.TaskSpec{
						DummyTaskSpec: v1alpha1.DummyTaskSpec{WantErr: false},
					},
				},
			},
			templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{
					{Name: "stepOne", Status: v1alpha1.ErrorStatus},
					{Name: "stepTwo", Status: v1alpha1.ExecutionInProgress},
				}}},
			},
			kustomizer: enhancer,
		},
		{name: "plan in progress with multiple parallel steps, will respect parallel step strategy and continue when first step fails", activePlan: &activePlan{
			name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{
					{Name: "stepOne", Status: v1alpha1.ExecutionInProgress},
					{Name: "stepTwo", Status: v1alpha1.ExecutionInProgress},
				}}},
			},
			spec: &v1alpha1.Plan{
				Strategy: "serial",
				Phases: []v1alpha1.Phase{
					{Name: "phase", Strategy: "parallel", Steps: []v1alpha1.Step{
						{Name: "stepOne", Tasks: []string{"taskOne"}},
						{Name: "stepTwo", Tasks: []string{"taskTwo"}},
					}},
				},
			},
			tasks: []v1alpha1.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: v1alpha1.TaskSpec{
						DummyTaskSpec: v1alpha1.DummyTaskSpec{WantErr: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: v1alpha1.TaskSpec{
						DummyTaskSpec: v1alpha1.DummyTaskSpec{WantErr: false},
					},
				},
			},
			templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{
					{Name: "stepOne", Status: v1alpha1.ErrorStatus},
					{Name: "stepTwo", Status: v1alpha1.ExecutionComplete},
				}}},
			},
			kustomizer: enhancer,
		},
		{name: "plan in progress with multiple parallel steps, will stop the execution on the first fatal step error", activePlan: &activePlan{
			name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{
					{Name: "stepOne", Status: v1alpha1.ExecutionInProgress},
					{Name: "stepTwo", Status: v1alpha1.ExecutionInProgress},
				}}},
			},
			spec: &v1alpha1.Plan{
				Strategy: "serial",
				Phases: []v1alpha1.Phase{
					{Name: "phase", Strategy: "parallel", Steps: []v1alpha1.Step{
						{Name: "stepOne", Tasks: []string{"taskOne"}},
						{Name: "stepTwo", Tasks: []string{"taskTwo"}},
					}},
				},
			},
			tasks: []v1alpha1.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: v1alpha1.TaskSpec{
						DummyTaskSpec: v1alpha1.DummyTaskSpec{WantErr: true, Fatal: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: v1alpha1.TaskSpec{
						DummyTaskSpec: v1alpha1.DummyTaskSpec{WantErr: false},
					},
				},
			},
			templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionFatalError,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionFatalError, Steps: []v1alpha1.StepStatus{
					{Name: "stepOne", Status: v1alpha1.ExecutionFatalError},
					{Name: "stepTwo", Status: v1alpha1.ExecutionInProgress},
				}}},
			},
			wantErr:    true,
			kustomizer: enhancer,
		},
		// --- Respect the Phases execution strategy ---
		{name: "plan in progress with multiple serial phases, will respect serial phase strategy and stop after first phase fails", activePlan: &activePlan{
			name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{
					{Name: "phaseOne", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Name: "step", Status: v1alpha1.ExecutionInProgress}}},
					{Name: "phaseTwo", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Name: "step", Status: v1alpha1.ExecutionInProgress}}},
				},
			},
			spec: &v1alpha1.Plan{
				Strategy: "serial",
				Phases: []v1alpha1.Phase{
					{Name: "phaseOne", Strategy: "serial", Steps: []v1alpha1.Step{{Name: "step", Tasks: []string{"taskOne"}}}},
					{Name: "phaseTwo", Strategy: "serial", Steps: []v1alpha1.Step{{Name: "step", Tasks: []string{"taskTwo"}}}},
				},
			},
			tasks: []v1alpha1.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: v1alpha1.TaskSpec{
						DummyTaskSpec: v1alpha1.DummyTaskSpec{WantErr: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: v1alpha1.TaskSpec{
						DummyTaskSpec: v1alpha1.DummyTaskSpec{WantErr: false},
					},
				},
			},
			templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{
					{Name: "phaseOne", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Name: "step", Status: v1alpha1.ErrorStatus}}},
					{Name: "phaseTwo", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Name: "step", Status: v1alpha1.ExecutionInProgress}}},
				},
			},
			kustomizer: enhancer,
		},
		{name: "plan in progress with multiple parallel phases, will respect parallel phase strategy and continue after first phase fails", activePlan: &activePlan{
			name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{
					{Name: "phaseOne", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Name: "step", Status: v1alpha1.ExecutionInProgress}}},
					{Name: "phaseTwo", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Name: "step", Status: v1alpha1.ExecutionInProgress}}},
				},
			},
			spec: &v1alpha1.Plan{
				Strategy: "parallel",
				Phases: []v1alpha1.Phase{
					{Name: "phaseOne", Strategy: "serial", Steps: []v1alpha1.Step{{Name: "step", Tasks: []string{"taskOne"}}}},
					{Name: "phaseTwo", Strategy: "serial", Steps: []v1alpha1.Step{{Name: "step", Tasks: []string{"taskTwo"}}}},
				},
			},
			tasks: []v1alpha1.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: v1alpha1.TaskSpec{
						DummyTaskSpec: v1alpha1.DummyTaskSpec{WantErr: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: v1alpha1.TaskSpec{
						DummyTaskSpec: v1alpha1.DummyTaskSpec{WantErr: false},
					},
				},
			},
			templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{
					{Name: "phaseOne", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Name: "step", Status: v1alpha1.ErrorStatus}}},
					{Name: "phaseTwo", Status: v1alpha1.ExecutionComplete, Steps: []v1alpha1.StepStatus{{Name: "step", Status: v1alpha1.ExecutionComplete}}},
				},
			},
			kustomizer: enhancer,
		},
		{name: "plan in progress with multiple parallel phases, will stop the execution on the first fatal step error", activePlan: &activePlan{
			name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{
					{Name: "phaseOne", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Name: "step", Status: v1alpha1.ExecutionInProgress}}},
					{Name: "phaseTwo", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Name: "step", Status: v1alpha1.ExecutionInProgress}}},
				},
			},
			spec: &v1alpha1.Plan{
				Strategy: "parallel",
				Phases: []v1alpha1.Phase{
					{Name: "phaseOne", Strategy: "serial", Steps: []v1alpha1.Step{{Name: "step", Tasks: []string{"taskOne"}}}},
					{Name: "phaseTwo", Strategy: "serial", Steps: []v1alpha1.Step{{Name: "step", Tasks: []string{"taskTwo"}}}},
				},
			},
			tasks: []v1alpha1.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: v1alpha1.TaskSpec{
						DummyTaskSpec: v1alpha1.DummyTaskSpec{WantErr: true, Fatal: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: v1alpha1.TaskSpec{
						DummyTaskSpec: v1alpha1.DummyTaskSpec{WantErr: false},
					},
				},
			},
			templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionFatalError,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{
					{Name: "phaseOne", Status: v1alpha1.ExecutionFatalError, Steps: []v1alpha1.StepStatus{{Name: "step", Status: v1alpha1.ExecutionFatalError}}},
					{Name: "phaseTwo", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Name: "step", Status: v1alpha1.ExecutionInProgress}}},
				},
			},
			wantErr:    true,
			kustomizer: enhancer,
		},
		{
			name: "plan with an enhancer that fails to kustomize the resources will result in a fatal failure",
			activePlan: &activePlan{
				name: "test",
				PlanStatus: &v1alpha1.PlanStatus{
					Status: v1alpha1.ExecutionPending,
					Name:   "test",
					Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionPending, Steps: []v1alpha1.StepStatus{{Name: "step", Status: v1alpha1.ExecutionPending}}}},
				},
				spec: &v1alpha1.Plan{
					Strategy: "serial",
					Phases: []v1alpha1.Phase{
						{Name: "phase", Strategy: "serial", Steps: []v1alpha1.Step{{Name: "step", Tasks: []string{"task"}}}},
					},
				},
				tasks: []v1alpha1.Task{
					{
						Name: "task",
						Kind: "Apply",
						Spec: v1alpha1.TaskSpec{
							ApplyTaskSpec: v1alpha1.ApplyTaskSpec{Resources: []string{"job"}},
						},
					},
				},
				templates: map[string]string{"job": getResourceAsString(job("job1", "default"))},
			},
			metadata: meta,
			expectedStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionFatalError,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionFatalError, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionFatalError, Name: "step"}}}}},
			wantErr:    true,
			kustomizer: errEnhancer,
		},
	}

	for _, tt := range tests {
		testClient := fake.NewFakeClientWithScheme(scheme.Scheme)
		newStatus, err := executePlan(tt.activePlan, tt.metadata, testClient, tt.kustomizer)

		if !tt.wantErr && err != nil {
			t.Errorf("%s: Expecting no error but got one: %v", tt.name, err)
		}

		if tt.wantErr && err == nil {
			t.Errorf("%s: Expecting an error but got none", tt.name)
		}

		if !reflect.DeepEqual(tt.expectedStatus, newStatus) {
			t.Errorf("%s: Expecting status to be:\n%v \nbut got:\n%v", tt.name, *tt.expectedStatus, *newStatus)
		}
	}
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

func instance() *v1alpha1.Instance {
	return &v1alpha1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1alpha1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
		},
		Spec: v1alpha1.InstanceSpec{
			OperatorVersion: corev1.ObjectReference{
				Name: "first-operator",
			},
		},
	}
}

func getResourceAsString(resource v1.Object) string {
	bytes, _ := yaml.Marshal(resource)
	return string(bytes)
}

type testKubernetesObjectEnhancer struct{}

func (k *testKubernetesObjectEnhancer) ApplyConventionsToTemplates(templates map[string]string, metadata task.ExecutionMetadata) ([]runtime.Object, error) {
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

func (k *errKubernetesObjectEnhancer) ApplyConventionsToTemplates(templates map[string]string, metadata task.ExecutionMetadata) ([]runtime.Object, error) {
	return nil, errors.New("always error")
}
