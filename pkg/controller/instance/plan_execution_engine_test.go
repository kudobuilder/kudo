package instance

import (
	"reflect"
	"testing"
	"time"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/engine/task"
	engtask "github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/util/template"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestExecutePlan(t *testing.T) {
	timeNow := time.Now()
	instance := instance()
	meta := &engtask.EngineMetadata{
		InstanceName:        instance.Name,
		InstanceNamespace:   instance.Namespace,
		OperatorName:        "first-operator",
		OperatorVersionName: "first-operator-1.0",
		OperatorVersion:     "1.0",
		ResourcesOwner:      instance,
	}
	testEnhancer := &testKubernetesObjectEnhancer{}

	tests := []struct {
		name           string
		activePlan     *activePlan
		metadata       *task.EngineMetadata
		expectedStatus *v1alpha1.PlanStatus
		wantErr        bool
		enhancer       task.KubernetesObjectEnhancer
	}{
		{name: "plan already finished will not change its status", activePlan: &activePlan{
			name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionComplete,
			},
		},
			metadata:       meta,
			expectedStatus: &v1alpha1.PlanStatus{Status: v1alpha1.ExecutionComplete},
			enhancer:       testEnhancer,
		},
		{name: "plan with a step to be executed is in progress when the step is not completed", activePlan: &activePlan{
			name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionInProgress, Name: "step"}}}},
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
						DummyTaskSpec: v1alpha1.DummyTaskSpec{Done: false},
					},
				},
			},
			templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionInProgress, Name: "step"}}}}},
			enhancer: testEnhancer,
		},
		{name: "plan with one step that is healthy is marked as completed", activePlan: &activePlan{
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
					Kind: "Dummy",
					Spec: v1alpha1.TaskSpec{
						DummyTaskSpec: v1alpha1.DummyTaskSpec{Done: true},
					},
				},
			},
			templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1alpha1.PlanStatus{
				Status:          v1alpha1.ExecutionComplete,
				LastFinishedRun: v1.Time{Time: timeNow},
				Name:            "test",
				Phases:          []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionComplete, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionComplete, Name: "step"}}}},
			},
			enhancer: testEnhancer,
		},
		{name: "plan in errored state will be retried and completed when the step is done", activePlan: &activePlan{
			name: "test",
			PlanStatus: &v1alpha1.PlanStatus{
				Status: v1alpha1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionInProgress, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ErrorStatus, Name: "step"}}}},
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
						DummyTaskSpec: v1alpha1.DummyTaskSpec{Done: true},
					},
				},
			},
			templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1alpha1.PlanStatus{
				Status:          v1alpha1.ExecutionComplete,
				LastFinishedRun: v1.Time{Time: timeNow},
				Name:            "test",
				Phases:          []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionComplete, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionComplete, Name: "step"}}}},
			},
			enhancer: testEnhancer,
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
			enhancer: testEnhancer,
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
			wantErr:  true,
			enhancer: testEnhancer,
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
			wantErr:  true,
			enhancer: testEnhancer,
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
			wantErr:  true,
			enhancer: testEnhancer,
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
			enhancer: testEnhancer,
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
						DummyTaskSpec: v1alpha1.DummyTaskSpec{Done: true},
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
			enhancer: testEnhancer,
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
			wantErr:  true,
			enhancer: testEnhancer,
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
			enhancer: testEnhancer,
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
						DummyTaskSpec: v1alpha1.DummyTaskSpec{Done: true},
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
			enhancer: testEnhancer,
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
			wantErr:  true,
			enhancer: testEnhancer,
		},
		{
			name: "plan in a pending status will have fatal plan/phase/step statuses when a step has a fatal error",
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
				Phases: []v1alpha1.PhaseStatus{{Name: "phase", Status: v1alpha1.ExecutionFatalError, Steps: []v1alpha1.StepStatus{{Status: v1alpha1.ExecutionFatalError, Name: "step"}}}}},
			wantErr:  true,
			enhancer: testEnhancer,
		},
	}

	for _, tt := range tests {
		testClient := fake.NewFakeClientWithScheme(scheme.Scheme)
		newStatus, err := executePlan(tt.activePlan, tt.metadata, testClient, tt.enhancer, timeNow)

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
