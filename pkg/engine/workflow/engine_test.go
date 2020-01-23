package workflow

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/engine/renderer"
)

func TestExecutePlan(t *testing.T) {
	timeNow := time.Now()
	instance := instance()
	meta := &engine.Metadata{
		InstanceName:        instance.Name,
		InstanceNamespace:   instance.Namespace,
		OperatorName:        "first-operator",
		OperatorVersionName: "first-operator-1.0",
		OperatorVersion:     "1.0",
		ResourcesOwner:      instance,
	}
	testEnhancer := &testEnhancer{}

	tests := []struct {
		name           string
		activePlan     *ActivePlan
		metadata       *engine.Metadata
		expectedStatus *v1beta1.PlanStatus
		wantErr        bool
		enhancer       renderer.Enhancer
	}{
		{name: "plan already finished will not change its status", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionComplete,
			},
		},
			metadata:       meta,
			expectedStatus: &v1beta1.PlanStatus{Status: v1beta1.ExecutionComplete},
			enhancer:       testEnhancer,
		},
		{name: "plan with a step to be executed is in progress when the step is not completed", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Status: v1beta1.ExecutionInProgress, Name: "step"}}}},
			},
			Spec: &v1beta1.Plan{
				Strategy: "serial",
				Phases: []v1beta1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1beta1.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			Tasks: []v1beta1.Task{
				{
					Name: "task",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{Done: false},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Status: v1beta1.ExecutionInProgress, Name: "step"}}}}},
			enhancer: testEnhancer,
		},
		{name: "plan with one step that is healthy is marked as completed", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionPending,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionPending, Steps: []v1beta1.StepStatus{{Status: v1beta1.ExecutionPending, Name: "step"}}}},
			},
			Spec: &v1beta1.Plan{
				Strategy: "serial",
				Phases: []v1beta1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1beta1.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			Tasks: []v1beta1.Task{
				{
					Name: "task",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{Done: true},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1beta1.PlanStatus{
				Status:          v1beta1.ExecutionComplete,
				LastFinishedRun: v1.Time{Time: timeNow},
				Name:            "test",
				Phases:          []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionComplete, Steps: []v1beta1.StepStatus{{Status: v1beta1.ExecutionComplete, Name: "step"}}}},
			},
			enhancer: testEnhancer,
		},
		{name: "plan in errored state will be retried and completed when the step is done", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Status: v1beta1.ErrorStatus, Name: "step"}}}},
			},
			Spec: &v1beta1.Plan{
				Strategy: "serial",
				Phases: []v1beta1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1beta1.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			Tasks: []v1beta1.Task{
				{
					Name: "task",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{Done: true},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1beta1.PlanStatus{
				Status:          v1beta1.ExecutionComplete,
				LastFinishedRun: v1.Time{Time: timeNow},
				Name:            "test",
				Phases:          []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionComplete, Steps: []v1beta1.StepStatus{{Status: v1beta1.ExecutionComplete, Name: "step"}}}},
			},
			enhancer: testEnhancer,
		},
		// --- Proper error and fatal error status propagation ---
		{name: "plan in progress, will have step error status, when a task fails", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Name: "step", Status: v1beta1.ExecutionInProgress}}}},
			},
			Spec: &v1beta1.Plan{
				Strategy: "serial",
				Phases: []v1beta1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1beta1.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			Tasks: []v1beta1.Task{
				{
					Name: "task",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{WantErr: true},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Status: v1beta1.ErrorStatus, Name: "step",
					Message: "A transient error when executing task test.phase.step.task. Will retry. dummy error"}}}},
			},
			enhancer: testEnhancer,
		},
		{name: "plan in progress, will have plan/phase/step fatal error status, when a task fails with a fatal error", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Name: "step", Status: v1beta1.ExecutionInProgress}}}},
			},
			Spec: &v1beta1.Plan{
				Strategy: "serial",
				Phases: []v1beta1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1beta1.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			Tasks: []v1beta1.Task{
				{
					Name: "task",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{WantErr: true, Fatal: true},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionFatalError,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionFatalError, Steps: []v1beta1.StepStatus{{Status: v1beta1.ExecutionFatalError,
					Message: "Error during execution: fatal error: default/test-instance failed in test.phase.step.task: dummy error", Name: "step"}}}},
			},
			wantErr:  true,
			enhancer: testEnhancer,
		},
		{name: "plan in progress with a misconfigured task will fail with a fatal error", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Name: "step", Status: v1beta1.ExecutionInProgress}}}},
			},
			Spec: &v1beta1.Plan{
				Strategy: "serial",
				Phases: []v1beta1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1beta1.Step{{Name: "step", Tasks: []string{"fake-task"}}}},
				},
			},
			Tasks: []v1beta1.Task{
				{
					Name: "task",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{WantErr: false},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionFatalError,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionFatalError, Steps: []v1beta1.StepStatus{{Status: v1beta1.ExecutionFatalError,
					Message: "default/test-instance fatal error:  missing task test.phase.step.fake-task", Name: "step"}}}},
			},
			wantErr:  true,
			enhancer: testEnhancer,
		},
		{name: "plan in progress with an unknown task spec will fail with a fatal error", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Name: "step", Status: v1beta1.ExecutionInProgress}}}},
			},
			Spec: &v1beta1.Plan{
				Strategy: "serial",
				Phases: []v1beta1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1beta1.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			Tasks: []v1beta1.Task{
				{
					Name: "task",
					Kind: "Unknown",
					Spec: v1beta1.TaskSpec{},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionFatalError,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionFatalError, Steps: []v1beta1.StepStatus{{Status: v1beta1.ExecutionFatalError, Name: "step",
					Message: "default/test-instance fatal error:  failed to build task test.phase.step.task: unknown task kind Unknown"}}}},
			},
			wantErr:  true,
			enhancer: testEnhancer,
		},
		// --- Respect the Steps execution strategy ---
		{name: "plan in progress with multiple serial steps, will respect serial step strategy and stop after first step fails", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{
					{Name: "stepOne", Status: v1beta1.ExecutionInProgress},
					{Name: "stepTwo", Status: v1beta1.ExecutionInProgress},
				}}},
			},
			Spec: &v1beta1.Plan{
				Strategy: "serial",
				Phases: []v1beta1.Phase{
					{Name: "phase", Strategy: "serial", Steps: []v1beta1.Step{
						{Name: "stepOne", Tasks: []string{"taskOne"}},
						{Name: "stepTwo", Tasks: []string{"taskTwo"}},
					}},
				},
			},
			Tasks: []v1beta1.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{WantErr: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{WantErr: false},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{
					{Name: "stepOne", Status: v1beta1.ErrorStatus, Message: "A transient error when executing task test.phase.stepOne.taskOne. Will retry. dummy error"},
					{Name: "stepTwo", Status: v1beta1.ExecutionInProgress},
				}}},
			},
			enhancer: testEnhancer,
		},
		{name: "plan in progress with multiple parallel steps, will respect parallel step strategy and continue when first step fails", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{
					{Name: "stepOne", Status: v1beta1.ExecutionInProgress},
					{Name: "stepTwo", Status: v1beta1.ExecutionInProgress},
				}}},
			},
			Spec: &v1beta1.Plan{
				Strategy: "serial",
				Phases: []v1beta1.Phase{
					{Name: "phase", Strategy: "parallel", Steps: []v1beta1.Step{
						{Name: "stepOne", Tasks: []string{"taskOne"}},
						{Name: "stepTwo", Tasks: []string{"taskTwo"}},
					}},
				},
			},
			Tasks: []v1beta1.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{WantErr: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{Done: true},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{
					{Name: "stepOne", Status: v1beta1.ErrorStatus, Message: "A transient error when executing task test.phase.stepOne.taskOne. Will retry. dummy error"},
					{Name: "stepTwo", Status: v1beta1.ExecutionComplete},
				}}},
			},
			enhancer: testEnhancer,
		},
		{name: "plan in progress with multiple parallel steps, will stop the execution on the first fatal step error", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{
					{Name: "stepOne", Status: v1beta1.ExecutionInProgress},
					{Name: "stepTwo", Status: v1beta1.ExecutionInProgress},
				}}},
			},
			Spec: &v1beta1.Plan{
				Strategy: "serial",
				Phases: []v1beta1.Phase{
					{Name: "phase", Strategy: "parallel", Steps: []v1beta1.Step{
						{Name: "stepOne", Tasks: []string{"taskOne"}},
						{Name: "stepTwo", Tasks: []string{"taskTwo"}},
					}},
				},
			},
			Tasks: []v1beta1.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{WantErr: true, Fatal: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{WantErr: false},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionFatalError,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionFatalError, Steps: []v1beta1.StepStatus{
					{Name: "stepOne", Status: v1beta1.ExecutionFatalError, Message: "Error during execution: fatal error: default/test-instance failed in test.phase.stepOne.taskOne: dummy error"},
					{Name: "stepTwo", Status: v1beta1.ExecutionInProgress},
				}}},
			},
			wantErr:  true,
			enhancer: testEnhancer,
		},
		// --- Respect the Phases execution strategy ---
		{name: "plan in progress with multiple serial phases, will respect serial phase strategy and stop after first phase fails", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{
					{Name: "phaseOne", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Name: "step", Status: v1beta1.ExecutionInProgress}}},
					{Name: "phaseTwo", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Name: "step", Status: v1beta1.ExecutionInProgress}}},
				},
			},
			Spec: &v1beta1.Plan{
				Strategy: "serial",
				Phases: []v1beta1.Phase{
					{Name: "phaseOne", Strategy: "serial", Steps: []v1beta1.Step{{Name: "step", Tasks: []string{"taskOne"}}}},
					{Name: "phaseTwo", Strategy: "serial", Steps: []v1beta1.Step{{Name: "step", Tasks: []string{"taskTwo"}}}},
				},
			},
			Tasks: []v1beta1.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{WantErr: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{WantErr: false},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{
					{Name: "phaseOne", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Name: "step", Status: v1beta1.ErrorStatus, Message: "A transient error when executing task test.phaseOne.step.taskOne. Will retry. dummy error"}}},
					{Name: "phaseTwo", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Name: "step", Status: v1beta1.ExecutionInProgress}}},
				},
			},
			enhancer: testEnhancer,
		},
		{name: "plan in progress with multiple parallel phases, will respect parallel phase strategy and continue after first phase fails", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{
					{Name: "phaseOne", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Name: "step", Status: v1beta1.ExecutionInProgress}}},
					{Name: "phaseTwo", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Name: "step", Status: v1beta1.ExecutionInProgress}}},
				},
			},
			Spec: &v1beta1.Plan{
				Strategy: "parallel",
				Phases: []v1beta1.Phase{
					{Name: "phaseOne", Strategy: "serial", Steps: []v1beta1.Step{{Name: "step", Tasks: []string{"taskOne"}}}},
					{Name: "phaseTwo", Strategy: "serial", Steps: []v1beta1.Step{{Name: "step", Tasks: []string{"taskTwo"}}}},
				},
			},
			Tasks: []v1beta1.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{WantErr: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{Done: true},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{
					{Name: "phaseOne", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Name: "step", Status: v1beta1.ErrorStatus, Message: "A transient error when executing task test.phaseOne.step.taskOne. Will retry. dummy error"}}},
					{Name: "phaseTwo", Status: v1beta1.ExecutionComplete, Steps: []v1beta1.StepStatus{{Name: "step", Status: v1beta1.ExecutionComplete}}},
				},
			},
			enhancer: testEnhancer,
		},
		{name: "plan in progress with multiple parallel phases, will stop the execution on the first fatal step error", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionInProgress,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{
					{Name: "phaseOne", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Name: "step", Status: v1beta1.ExecutionInProgress}}},
					{Name: "phaseTwo", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Name: "step", Status: v1beta1.ExecutionInProgress}}},
				},
			},
			Spec: &v1beta1.Plan{
				Strategy: "parallel",
				Phases: []v1beta1.Phase{
					{Name: "phaseOne", Strategy: "serial", Steps: []v1beta1.Step{{Name: "step", Tasks: []string{"taskOne"}}}},
					{Name: "phaseTwo", Strategy: "serial", Steps: []v1beta1.Step{{Name: "step", Tasks: []string{"taskTwo"}}}},
				},
			},
			Tasks: []v1beta1.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{WantErr: true, Fatal: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{WantErr: false},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionFatalError,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{
					{Name: "phaseOne", Status: v1beta1.ExecutionFatalError, Steps: []v1beta1.StepStatus{{Name: "step", Status: v1beta1.ExecutionFatalError,
						Message: "Error during execution: fatal error: default/test-instance failed in test.phaseOne.step.taskOne: dummy error"}}},
					{Name: "phaseTwo", Status: v1beta1.ExecutionInProgress, Steps: []v1beta1.StepStatus{{Name: "step", Status: v1beta1.ExecutionInProgress}}},
				},
			},
			wantErr:  true,
			enhancer: testEnhancer,
		},
		{
			name: "plan in a pending status will have fatal plan/phase/step statuses when a step has a fatal error",
			activePlan: &ActivePlan{
				Name: "test",
				PlanStatus: &v1beta1.PlanStatus{
					Status: v1beta1.ExecutionPending,
					Name:   "test",
					Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionPending, Steps: []v1beta1.StepStatus{{Name: "step", Status: v1beta1.ExecutionPending}}}},
				},
				Spec: &v1beta1.Plan{
					Strategy: "serial",
					Phases: []v1beta1.Phase{
						{Name: "phase", Strategy: "serial", Steps: []v1beta1.Step{{Name: "step", Tasks: []string{"task"}}}},
					},
				},
				Tasks: []v1beta1.Task{
					{
						Name: "task",
						Kind: "Dummy",
						Spec: v1beta1.TaskSpec{
							DummyTaskSpec: v1beta1.DummyTaskSpec{WantErr: true, Fatal: true},
						},
					},
				},
				Templates: map[string]string{},
			},
			metadata: meta,
			expectedStatus: &v1beta1.PlanStatus{
				Status: v1beta1.ExecutionFatalError,
				Name:   "test",
				Phases: []v1beta1.PhaseStatus{{Name: "phase", Status: v1beta1.ExecutionFatalError, Steps: []v1beta1.StepStatus{{Status: v1beta1.ExecutionFatalError, Name: "step",
					Message: "Error during execution: fatal error: default/test-instance failed in test.phase.step.task: dummy error"}}}}},
			wantErr:  true,
			enhancer: testEnhancer,
		},
	}

	for _, tt := range tests {
		testClient := fake.NewFakeClientWithScheme(scheme.Scheme)
		newStatus, err := Execute(tt.activePlan, tt.metadata, testClient, tt.enhancer, timeNow)

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

func instance() *v1beta1.Instance {
	return &v1beta1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
		},
		Spec: v1beta1.InstanceSpec{
			OperatorVersion: corev1.ObjectReference{
				Name: "first-operator",
			},
		},
	}
}

type testEnhancer struct{}

func (k *testEnhancer) Apply(templates map[string]string, metadata renderer.Metadata) ([]runtime.Object, error) {
	result := make([]runtime.Object, 0)
	for _, t := range templates {
		objsToAdd, err := renderer.YamlToObject(t)
		if err != nil {
			return nil, fmt.Errorf("error parsing kubernetes objects after applying enhance: %v", err)
		}
		result = append(result, objsToAdd[0])
	}
	return result, nil
}
