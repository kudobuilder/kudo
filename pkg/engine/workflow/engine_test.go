package workflow

import (
	"reflect"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/engine/renderer"
	kudofake "github.com/kudobuilder/kudo/pkg/test/fake"
)

var testTime = time.Date(2019, 10, 17, 1, 1, 1, 1, time.UTC)

func TestExecutePlan(t *testing.T) {
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
		expectedStatus *kudoapi.PlanStatus
		wantErr        bool
		enhancer       renderer.Enhancer
	}{
		{name: "plan already finished will not change its status", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &kudoapi.PlanStatus{
				Status: kudoapi.ExecutionComplete,
			},
		},
			metadata:       meta,
			expectedStatus: &kudoapi.PlanStatus{Status: kudoapi.ExecutionComplete, LastUpdatedTimestamp: &v1.Time{Time: testTime}},
			enhancer:       testEnhancer,
		},
		{name: "plan with a step to be executed is in progress when the step is not completed", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &kudoapi.PlanStatus{
				Name:                 "test",
				Status:               kudoapi.ExecutionInProgress,
				LastUpdatedTimestamp: &v1.Time{Time: testTime},
				Phases:               []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Status: kudoapi.ExecutionInProgress, Name: "step"}}}},
			},
			Spec: &kudoapi.Plan{
				Strategy: "serial",
				Phases: []kudoapi.Phase{
					{Name: "phase", Strategy: "serial", Steps: []kudoapi.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			Tasks: []kudoapi.Task{
				{
					Name: "task",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{Done: false},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &kudoapi.PlanStatus{
				Name:                 "test",
				Status:               kudoapi.ExecutionInProgress,
				LastUpdatedTimestamp: &v1.Time{Time: testTime},
				Phases:               []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Status: kudoapi.ExecutionInProgress, Name: "step"}}}}},
			enhancer: testEnhancer,
		},
		{name: "plan with one step that is healthy is marked as completed", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &kudoapi.PlanStatus{
				Name:                 "test",
				Status:               kudoapi.ExecutionPending,
				LastUpdatedTimestamp: &v1.Time{Time: testTime},
				Phases:               []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionPending, Steps: []kudoapi.StepStatus{{Status: kudoapi.ExecutionPending, Name: "step"}}}},
			},
			Spec: &kudoapi.Plan{
				Strategy: "serial",
				Phases: []kudoapi.Phase{
					{Name: "phase", Strategy: "serial", Steps: []kudoapi.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			Tasks: []kudoapi.Task{
				{
					Name: "task",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{Done: true},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &kudoapi.PlanStatus{
				Name:                 "test",
				Status:               kudoapi.ExecutionComplete,
				LastUpdatedTimestamp: &v1.Time{Time: testTime},
				Phases:               []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionComplete, Steps: []kudoapi.StepStatus{{Status: kudoapi.ExecutionComplete, Name: "step"}}}},
			},
			enhancer: testEnhancer,
		},
		{name: "plan in errored state will be retried and completed when the step is done", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &kudoapi.PlanStatus{
				Name:                 "test",
				LastUpdatedTimestamp: &v1.Time{Time: testTime},
				Phases:               []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Status: kudoapi.ErrorStatus, Name: "step"}}}},
			},
			Spec: &kudoapi.Plan{
				Strategy: "serial",
				Phases: []kudoapi.Phase{
					{Name: "phase", Strategy: "serial", Steps: []kudoapi.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			Tasks: []kudoapi.Task{
				{
					Name: "task",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{Done: true},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &kudoapi.PlanStatus{
				Name:                 "test",
				Status:               kudoapi.ExecutionComplete,
				LastUpdatedTimestamp: &v1.Time{Time: testTime},
				Phases:               []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionComplete, Steps: []kudoapi.StepStatus{{Status: kudoapi.ExecutionComplete, Name: "step"}}}},
			},
			enhancer: testEnhancer,
		},
		// --- Proper error and fatal error status propagation ---
		{name: "plan in progress, will have step error status, when a task fails", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &kudoapi.PlanStatus{
				Name:   "test",
				Status: kudoapi.ExecutionInProgress,
				Phases: []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Name: "step", Status: kudoapi.ExecutionInProgress}}}},
			},
			Spec: &kudoapi.Plan{
				Strategy: "serial",
				Phases: []kudoapi.Phase{
					{Name: "phase", Strategy: "serial", Steps: []kudoapi.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			Tasks: []kudoapi.Task{
				{
					Name: "task",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{WantErr: true},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &kudoapi.PlanStatus{
				Name:                 "test",
				Status:               kudoapi.ExecutionInProgress,
				LastUpdatedTimestamp: &v1.Time{Time: testTime},
				Phases: []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Status: kudoapi.ErrorStatus, Name: "step",
					Message: "A transient error when executing task test.phase.step.task. Will retry. dummy error"}}}},
			},
			enhancer: testEnhancer,
		},
		{name: "plan in progress, will have plan/phase/step fatal error status, when a task fails with a fatal error", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &kudoapi.PlanStatus{
				Name:   "test",
				Status: kudoapi.ExecutionInProgress,
				Phases: []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Name: "step", Status: kudoapi.ExecutionInProgress}}}},
			},
			Spec: &kudoapi.Plan{
				Strategy: "serial",
				Phases: []kudoapi.Phase{
					{Name: "phase", Strategy: "serial", Steps: []kudoapi.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			Tasks: []kudoapi.Task{
				{
					Name: "task",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{WantErr: true, Fatal: true},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &kudoapi.PlanStatus{
				Name:                 "test",
				Status:               kudoapi.ExecutionFatalError,
				LastUpdatedTimestamp: &v1.Time{Time: testTime},
				Phases: []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionFatalError, Steps: []kudoapi.StepStatus{{Status: kudoapi.ExecutionFatalError,
					Message: "Error during execution: fatal error: default/test-instance failed in test.phase.step.task: dummy error", Name: "step"}}}},
			},
			wantErr:  true,
			enhancer: testEnhancer,
		},
		{name: "plan in progress with a misconfigured task will fail with a fatal error", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &kudoapi.PlanStatus{
				Name:   "test",
				Status: kudoapi.ExecutionInProgress,
				Phases: []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Name: "step", Status: kudoapi.ExecutionInProgress}}}},
			},
			Spec: &kudoapi.Plan{
				Strategy: "serial",
				Phases: []kudoapi.Phase{
					{Name: "phase", Strategy: "serial", Steps: []kudoapi.Step{{Name: "step", Tasks: []string{"fake-task"}}}},
				},
			},
			Tasks: []kudoapi.Task{
				{
					Name: "task",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{WantErr: false},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &kudoapi.PlanStatus{
				Name:                 "test",
				Status:               kudoapi.ExecutionFatalError,
				LastUpdatedTimestamp: &v1.Time{Time: testTime},
				Phases: []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionFatalError, Steps: []kudoapi.StepStatus{{Status: kudoapi.ExecutionFatalError,
					Message: "default/test-instance fatal error:  missing task test.phase.step.fake-task", Name: "step"}}}},
			},
			wantErr:  true,
			enhancer: testEnhancer,
		},
		{name: "plan in progress with an unknown task spec will fail with a fatal error", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &kudoapi.PlanStatus{
				Name:   "test",
				Status: kudoapi.ExecutionInProgress,
				Phases: []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Name: "step", Status: kudoapi.ExecutionInProgress}}}},
			},
			Spec: &kudoapi.Plan{
				Strategy: "serial",
				Phases: []kudoapi.Phase{
					{Name: "phase", Strategy: "serial", Steps: []kudoapi.Step{{Name: "step", Tasks: []string{"task"}}}},
				},
			},
			Tasks: []kudoapi.Task{
				{
					Name: "task",
					Kind: "Unknown",
					Spec: kudoapi.TaskSpec{},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &kudoapi.PlanStatus{
				Name:                 "test",
				Status:               kudoapi.ExecutionFatalError,
				LastUpdatedTimestamp: &v1.Time{Time: testTime},
				Phases: []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionFatalError, Steps: []kudoapi.StepStatus{{Status: kudoapi.ExecutionFatalError, Name: "step",
					Message: "default/test-instance fatal error:  failed to build task test.phase.step.task: unknown task kind Unknown"}}}},
			},
			wantErr:  true,
			enhancer: testEnhancer,
		},
		// --- Respect the Steps execution strategy ---
		{name: "plan in progress with multiple serial steps, will respect serial step strategy and stop after first step fails", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &kudoapi.PlanStatus{
				Name:   "test",
				Status: kudoapi.ExecutionInProgress,
				Phases: []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{
					{Name: "stepOne", Status: kudoapi.ExecutionInProgress},
					{Name: "stepTwo", Status: kudoapi.ExecutionInProgress},
				}}},
			},
			Spec: &kudoapi.Plan{
				Strategy: "serial",
				Phases: []kudoapi.Phase{
					{Name: "phase", Strategy: "serial", Steps: []kudoapi.Step{
						{Name: "stepOne", Tasks: []string{"taskOne"}},
						{Name: "stepTwo", Tasks: []string{"taskTwo"}},
					}},
				},
			},
			Tasks: []kudoapi.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{WantErr: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{WantErr: false},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &kudoapi.PlanStatus{
				Name:                 "test",
				Status:               kudoapi.ExecutionInProgress,
				LastUpdatedTimestamp: &v1.Time{Time: testTime},
				Phases: []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{
					{Name: "stepOne", Status: kudoapi.ErrorStatus, Message: "A transient error when executing task test.phase.stepOne.taskOne. Will retry. dummy error"},
					{Name: "stepTwo", Status: kudoapi.ExecutionInProgress},
				}}},
			},
			enhancer: testEnhancer,
		},
		{name: "plan in progress with multiple parallel steps, will respect parallel step strategy and continue when first step fails", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &kudoapi.PlanStatus{
				Name:   "test",
				Status: kudoapi.ExecutionInProgress,
				Phases: []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{
					{Name: "stepOne", Status: kudoapi.ExecutionInProgress},
					{Name: "stepTwo", Status: kudoapi.ExecutionInProgress},
				}}},
			},
			Spec: &kudoapi.Plan{
				Strategy: "serial",
				Phases: []kudoapi.Phase{
					{Name: "phase", Strategy: "parallel", Steps: []kudoapi.Step{
						{Name: "stepOne", Tasks: []string{"taskOne"}},
						{Name: "stepTwo", Tasks: []string{"taskTwo"}},
					}},
				},
			},
			Tasks: []kudoapi.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{WantErr: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{Done: true},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &kudoapi.PlanStatus{
				Name:                 "test",
				Status:               kudoapi.ExecutionInProgress,
				LastUpdatedTimestamp: &v1.Time{Time: testTime},
				Phases: []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{
					{Name: "stepOne", Status: kudoapi.ErrorStatus, Message: "A transient error when executing task test.phase.stepOne.taskOne. Will retry. dummy error"},
					{Name: "stepTwo", Status: kudoapi.ExecutionComplete},
				}}},
			},
			enhancer: testEnhancer,
		},
		{name: "plan in progress with multiple parallel steps, will stop the execution on the first fatal step error", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &kudoapi.PlanStatus{
				Status: kudoapi.ExecutionInProgress,
				Name:   "test",
				Phases: []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{
					{Name: "stepOne", Status: kudoapi.ExecutionInProgress},
					{Name: "stepTwo", Status: kudoapi.ExecutionInProgress},
				}}},
			},
			Spec: &kudoapi.Plan{
				Strategy: "serial",
				Phases: []kudoapi.Phase{
					{Name: "phase", Strategy: "parallel", Steps: []kudoapi.Step{
						{Name: "stepOne", Tasks: []string{"taskOne"}},
						{Name: "stepTwo", Tasks: []string{"taskTwo"}},
					}},
				},
			},
			Tasks: []kudoapi.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{WantErr: true, Fatal: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{WantErr: false},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &kudoapi.PlanStatus{
				Name:                 "test",
				Status:               kudoapi.ExecutionFatalError,
				LastUpdatedTimestamp: &v1.Time{Time: testTime},
				Phases: []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionFatalError, Steps: []kudoapi.StepStatus{
					{Name: "stepOne", Status: kudoapi.ExecutionFatalError, Message: "Error during execution: fatal error: default/test-instance failed in test.phase.stepOne.taskOne: dummy error"},
					{Name: "stepTwo", Status: kudoapi.ExecutionInProgress},
				}}},
			},
			wantErr:  true,
			enhancer: testEnhancer,
		},
		// --- Respect the Phases execution strategy ---
		{name: "plan in progress with multiple serial phases, will respect serial phase strategy and stop after first phase fails", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &kudoapi.PlanStatus{
				Name:   "test",
				Status: kudoapi.ExecutionInProgress,
				Phases: []kudoapi.PhaseStatus{
					{Name: "phaseOne", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Name: "step", Status: kudoapi.ExecutionInProgress}}},
					{Name: "phaseTwo", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Name: "step", Status: kudoapi.ExecutionInProgress}}},
				},
			},
			Spec: &kudoapi.Plan{
				Strategy: "serial",
				Phases: []kudoapi.Phase{
					{Name: "phaseOne", Strategy: "serial", Steps: []kudoapi.Step{{Name: "step", Tasks: []string{"taskOne"}}}},
					{Name: "phaseTwo", Strategy: "serial", Steps: []kudoapi.Step{{Name: "step", Tasks: []string{"taskTwo"}}}},
				},
			},
			Tasks: []kudoapi.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{WantErr: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{WantErr: false},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &kudoapi.PlanStatus{
				Name:                 "test",
				Status:               kudoapi.ExecutionInProgress,
				LastUpdatedTimestamp: &v1.Time{Time: testTime},
				Phases: []kudoapi.PhaseStatus{
					{Name: "phaseOne", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Name: "step", Status: kudoapi.ErrorStatus, Message: "A transient error when executing task test.phaseOne.step.taskOne. Will retry. dummy error"}}},
					{Name: "phaseTwo", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Name: "step", Status: kudoapi.ExecutionInProgress}}},
				},
			},
			enhancer: testEnhancer,
		},
		{name: "plan in progress with multiple parallel phases, will respect parallel phase strategy and continue after first phase fails", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &kudoapi.PlanStatus{
				Status: kudoapi.ExecutionInProgress,
				Name:   "test",
				Phases: []kudoapi.PhaseStatus{
					{Name: "phaseOne", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Name: "step", Status: kudoapi.ExecutionInProgress}}},
					{Name: "phaseTwo", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Name: "step", Status: kudoapi.ExecutionInProgress}}},
				},
			},
			Spec: &kudoapi.Plan{
				Strategy: "parallel",
				Phases: []kudoapi.Phase{
					{Name: "phaseOne", Strategy: "serial", Steps: []kudoapi.Step{{Name: "step", Tasks: []string{"taskOne"}}}},
					{Name: "phaseTwo", Strategy: "serial", Steps: []kudoapi.Step{{Name: "step", Tasks: []string{"taskTwo"}}}},
				},
			},
			Tasks: []kudoapi.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{WantErr: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{Done: true},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &kudoapi.PlanStatus{
				Name:                 "test",
				Status:               kudoapi.ExecutionInProgress,
				LastUpdatedTimestamp: &v1.Time{Time: testTime},
				Phases: []kudoapi.PhaseStatus{
					{Name: "phaseOne", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Name: "step", Status: kudoapi.ErrorStatus, Message: "A transient error when executing task test.phaseOne.step.taskOne. Will retry. dummy error"}}},
					{Name: "phaseTwo", Status: kudoapi.ExecutionComplete, Steps: []kudoapi.StepStatus{{Name: "step", Status: kudoapi.ExecutionComplete}}},
				},
			},
			enhancer: testEnhancer,
		},
		{name: "plan in progress with multiple parallel phases, will stop the execution on the first fatal step error", activePlan: &ActivePlan{
			Name: "test",
			PlanStatus: &kudoapi.PlanStatus{
				Name:   "test",
				Status: kudoapi.ExecutionInProgress,
				Phases: []kudoapi.PhaseStatus{
					{Name: "phaseOne", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Name: "step", Status: kudoapi.ExecutionInProgress}}},
					{Name: "phaseTwo", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Name: "step", Status: kudoapi.ExecutionInProgress}}},
				},
			},
			Spec: &kudoapi.Plan{
				Strategy: "parallel",
				Phases: []kudoapi.Phase{
					{Name: "phaseOne", Strategy: "serial", Steps: []kudoapi.Step{{Name: "step", Tasks: []string{"taskOne"}}}},
					{Name: "phaseTwo", Strategy: "serial", Steps: []kudoapi.Step{{Name: "step", Tasks: []string{"taskTwo"}}}},
				},
			},
			Tasks: []kudoapi.Task{
				{
					Name: "taskOne",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{WantErr: true, Fatal: true},
					},
				},
				{
					Name: "taskTwo",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{WantErr: false},
					},
				},
			},
			Templates: map[string]string{},
		},
			metadata: meta,
			expectedStatus: &kudoapi.PlanStatus{
				Name:                 "test",
				Status:               kudoapi.ExecutionFatalError,
				LastUpdatedTimestamp: &v1.Time{Time: testTime},
				Phases: []kudoapi.PhaseStatus{
					{Name: "phaseOne", Status: kudoapi.ExecutionFatalError, Steps: []kudoapi.StepStatus{{Name: "step", Status: kudoapi.ExecutionFatalError,
						Message: "Error during execution: fatal error: default/test-instance failed in test.phaseOne.step.taskOne: dummy error"}}},
					{Name: "phaseTwo", Status: kudoapi.ExecutionInProgress, Steps: []kudoapi.StepStatus{{Name: "step", Status: kudoapi.ExecutionInProgress}}},
				},
			},
			wantErr:  true,
			enhancer: testEnhancer,
		},
		{
			name: "plan in a pending status will have fatal plan/phase/step statuses when a step has a fatal error",
			activePlan: &ActivePlan{
				Name: "test",
				PlanStatus: &kudoapi.PlanStatus{
					Name:   "test",
					Status: kudoapi.ExecutionPending,
					Phases: []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionPending, Steps: []kudoapi.StepStatus{{Name: "step", Status: kudoapi.ExecutionPending}}}},
				},
				Spec: &kudoapi.Plan{
					Strategy: "serial",
					Phases: []kudoapi.Phase{
						{Name: "phase", Strategy: "serial", Steps: []kudoapi.Step{{Name: "step", Tasks: []string{"task"}}}},
					},
				},
				Tasks: []kudoapi.Task{
					{
						Name: "task",
						Kind: "Dummy",
						Spec: kudoapi.TaskSpec{
							DummyTaskSpec: kudoapi.DummyTaskSpec{WantErr: true, Fatal: true},
						},
					},
				},
				Templates: map[string]string{},
			},
			metadata: meta,
			expectedStatus: &kudoapi.PlanStatus{
				Name:                 "test",
				Status:               kudoapi.ExecutionFatalError,
				LastUpdatedTimestamp: &v1.Time{Time: testTime},
				Phases: []kudoapi.PhaseStatus{{Name: "phase", Status: kudoapi.ExecutionFatalError, Steps: []kudoapi.StepStatus{{Status: kudoapi.ExecutionFatalError, Name: "step",
					Message: "Error during execution: fatal error: default/test-instance failed in test.phase.step.task: dummy error"}}}}},
			wantErr:  true,
			enhancer: testEnhancer,
		},
	}

	testScheme := scheme.Scheme
	testClient := fake.NewFakeClientWithScheme(scheme.Scheme)
	fakeDiscovery := kudofake.CachedDiscoveryClient()
	fakeCachedDiscovery := memory.NewMemCacheClient(fakeDiscovery)
	for _, tt := range tests {
		newStatus, err := Execute(tt.activePlan, tt.metadata, testClient, fakeCachedDiscovery, nil, testScheme)
		newStatus.LastUpdatedTimestamp = &v1.Time{Time: testTime}

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

func instance() *kudoapi.Instance {
	return &kudoapi.Instance{
		TypeMeta: v1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-instance",
			Namespace: "default",
		},
		Spec: kudoapi.InstanceSpec{
			OperatorVersion: corev1.ObjectReference{
				Name: "first-operator",
			},
		},
	}
}

type testEnhancer struct{}

func (k *testEnhancer) Apply(objs []runtime.Object, metadata renderer.Metadata) ([]runtime.Object, error) {
	return objs, nil
}
