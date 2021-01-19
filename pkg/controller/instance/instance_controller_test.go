package instance

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/event"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/engine/task"
)

func Test_makePipes(t *testing.T) {
	meta := &engine.Metadata{
		InstanceName:        "first-operator-instance",
		InstanceNamespace:   "default",
		OperatorName:        "first-operator",
		OperatorVersionName: "first-operator-1.0",
		OperatorVersion:     "1.0",
	}

	tests := []struct {
		name     string
		planName string
		plan     *kudoapi.Plan
		tasks    []kudoapi.Task
		emeta    *engine.Metadata
		want     map[string]string
		wantErr  bool
	}{
		{
			name:     "no tasks, no pipes",
			planName: "deploy",
			plan: &kudoapi.Plan{Strategy: "serial", Phases: []kudoapi.Phase{
				{
					Name: "phase", Strategy: "serial", Steps: []kudoapi.Step{
						{
							Name: "step", Tasks: []string{}},
					}},
			}},
			tasks: []kudoapi.Task{},
			emeta: meta,
			want:  map[string]string{},
		},
		{
			name:     "no pipe tasks, no pipes",
			planName: "deploy",
			plan: &kudoapi.Plan{Strategy: "serial", Phases: []kudoapi.Phase{
				{
					Name: "phase", Strategy: "serial", Steps: []kudoapi.Step{
						{
							Name: "step", Tasks: []string{"task"}},
					}},
			}},
			tasks: []kudoapi.Task{
				{
					Name: "task",
					Kind: "Dummy",
					Spec: kudoapi.TaskSpec{
						DummyTaskSpec: kudoapi.DummyTaskSpec{Done: false},
					},
				},
			},
			emeta: meta,
			want:  map[string]string{},
		},
		{
			name:     "one pipe task, one pipes element",
			planName: "deploy",
			plan: &kudoapi.Plan{Strategy: "serial", Phases: []kudoapi.Phase{
				{
					Name: "phase", Strategy: "serial", Steps: []kudoapi.Step{
						{
							Name: "step", Tasks: []string{"task"}},
					}},
			}},
			tasks: []kudoapi.Task{
				{
					Name: "task",
					Kind: "Pipe",
					Spec: kudoapi.TaskSpec{
						PipeTaskSpec: kudoapi.PipeTaskSpec{
							Pod: "pipe-pod.yaml",
							Pipe: []kudoapi.PipeSpec{
								{
									File: "foo.txt",
									Kind: "Secret",
									Key:  "Foo",
								},
							},
						},
					},
				},
			},
			emeta: meta,
			want:  map[string]string{"Foo": "firstoperatorinstance.deploy.phase.step.task.foo"},
		},
		{
			name:     "two pipe tasks, two pipes element",
			planName: "deploy",
			plan: &kudoapi.Plan{Strategy: "serial", Phases: []kudoapi.Phase{
				{
					Name: "phase", Strategy: "serial", Steps: []kudoapi.Step{
						{Name: "stepOne", Tasks: []string{"task-one"}},
						{Name: "stepTwo", Tasks: []string{"task-two"}},
					}},
			}},
			tasks: []kudoapi.Task{
				{
					Name: "task-one",
					Kind: "Pipe",
					Spec: kudoapi.TaskSpec{
						PipeTaskSpec: kudoapi.PipeTaskSpec{
							Pod: "pipe-pod.yaml",
							Pipe: []kudoapi.PipeSpec{
								{
									File: "foo.txt",
									Kind: "Secret",
									Key:  "Foo",
								},
							},
						},
					},
				},
				{
					Name: "task-two",
					Kind: "Pipe",
					Spec: kudoapi.TaskSpec{
						PipeTaskSpec: kudoapi.PipeTaskSpec{
							Pod: "pipe-pod.yaml",
							Pipe: []kudoapi.PipeSpec{
								{
									File: "bar.txt",
									Kind: "ConfigMap",
									Key:  "Bar",
								},
							},
						},
					},
				},
			},
			emeta: meta,
			want: map[string]string{
				"Foo": "firstoperatorinstance.deploy.phase.stepone.taskone.foo",
				"Bar": "firstoperatorinstance.deploy.phase.steptwo.tasktwo.bar",
			},
		},
		{
			name:     "one pipe task, duplicated pipe keys",
			planName: "deploy",
			plan: &kudoapi.Plan{Strategy: "serial", Phases: []kudoapi.Phase{
				{
					Name: "phase", Strategy: "serial", Steps: []kudoapi.Step{
						{
							Name: "step", Tasks: []string{"task"}},
					}},
			}},
			tasks: []kudoapi.Task{
				{
					Name: "task",
					Kind: "Pipe",
					Spec: kudoapi.TaskSpec{
						PipeTaskSpec: kudoapi.PipeTaskSpec{
							Pod: "pipe-pod.yaml",
							Pipe: []kudoapi.PipeSpec{
								{
									File: "foo.txt",
									Kind: "Secret",
									Key:  "Foo",
								},
								{
									File: "bar.txt",
									Kind: "ConfigMap",
									Key:  "Foo",
								},
							},
						},
					},
				},
			},
			emeta:   meta,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			got, err := PipesMap(tt.planName, tt.plan, tt.tasks, tt.emeta)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("PipesMap() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParameterDiff(t *testing.T) {
	var (
		tests = []struct {
			name string
			new  map[string]string
			diff map[string]string
		}{
			{name: "update one value", new: map[string]string{"one": "11", "two": "2"}, diff: map[string]string{"one": "11"}},
			{name: "update multiple values", new: map[string]string{"one": "11", "two": "22"}, diff: map[string]string{"one": "11", "two": "22"}},
			{name: "add new value", new: map[string]string{"one": "1", "two": "2", "three": "3"}, diff: map[string]string{"three": "3"}},
			{name: "remove one value", new: map[string]string{"one": "1"}, diff: map[string]string{"two": "2"}},
			{name: "no difference", new: map[string]string{"one": "1", "two": "2"}, diff: map[string]string{}},
			{name: "empty new map", new: map[string]string{}, diff: map[string]string{"one": "1", "two": "2"}},
		}
	)

	var old = map[string]string{"one": "1", "two": "2"}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			diff := kudoapi.ParameterDiff(old, tt.new)
			assert.Equal(t, tt.diff, diff)
		})
	}
}

func TestRichParameterDiff(t *testing.T) {
	var empty = map[string]string{}
	var old = map[string]string{"one": "1", "two": "2"}

	var tests = []struct {
		name    string
		new     map[string]string
		changed map[string]string
		removed map[string]string
	}{
		{name: "update one value", new: map[string]string{"one": "11", "two": "2"}, changed: map[string]string{"one": "11"}, removed: empty},
		{name: "update multiple values", new: map[string]string{"one": "11", "two": "22"}, changed: map[string]string{"one": "11", "two": "22"}, removed: empty},
		{name: "add new value", new: map[string]string{"one": "1", "two": "2", "three": "3"}, changed: map[string]string{"three": "3"}, removed: empty},
		{name: "remove one value", new: map[string]string{"one": "1"}, changed: empty, removed: map[string]string{"two": "2"}},
		{name: "no difference", new: map[string]string{"one": "1", "two": "2"}, changed: empty, removed: empty},
		{name: "empty new map", new: empty, changed: empty, removed: map[string]string{"one": "1", "two": "2"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			changed, removed := kudoapi.RichParameterDiff(old, tt.new)
			assert.Equal(t, tt.changed, changed, "unexpected difference in changed parameters")
			assert.Equal(t, tt.removed, removed, "unexpected difference in removed parameters")
		})
	}
}

func TestEventFilterForDelete(t *testing.T) {
	var testParams = []struct {
		name    string
		allowed bool
		e       event.DeleteEvent
	}{
		{"A Pod without annotations", true, event.DeleteEvent{
			Meta:               &v1.Pod{},
			Object:             nil,
			DeleteStateUnknown: false,
		}},
		{"A Pod with pipePod annotation", false, event.DeleteEvent{
			Meta: &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{task.PipePodAnnotation: "true"},
				},
			},
			Object:             nil,
			DeleteStateUnknown: false,
		}},
	}

	filter := eventFilter()
	for _, test := range testParams {
		diff := filter.Delete(test.e)
		assert.Equal(t, test.allowed, diff, test.name)
	}
}

func Test_scheduledPlan(t *testing.T) {
	ov := &kudoapi.OperatorVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-operator", Namespace: "default"},
		TypeMeta:   metav1.TypeMeta{Kind: "OperatorVersion", APIVersion: "kudo.dev/v1beta1"},
		Spec:       kudoapi.OperatorVersionSpec{Plans: map[string]kudoapi.Plan{"cleanup": {}}},
	}

	idle := &kudoapi.Instance{
		TypeMeta:   metav1.TypeMeta{APIVersion: "kudo.dev/v1beta1", Kind: "Instance"},
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
		Spec: kudoapi.InstanceSpec{
			PlanExecution: kudoapi.PlanExecution{
				PlanName: "",
				UID:      "",
			},
		},
	}

	tests := []struct {
		name     string
		i        *kudoapi.Instance
		ov       *kudoapi.OperatorVersion
		wantPlan string
		wantUID  types.UID
	}{
		{
			name:     "idle instance return an empty plan and uid",
			i:        idle,
			ov:       ov,
			wantPlan: "",
			wantUID:  "",
		},
		{
			name: "scheduled instance return the scheduled plan and uid",
			i: func() *kudoapi.Instance {
				i := idle.DeepCopy()
				i.Spec.PlanExecution.PlanName = "deploy"
				i.Spec.PlanExecution.UID = "111-222-333-444"
				return i
			}(),
			ov:       ov,
			wantPlan: "deploy",
			wantUID:  "111-222-333-444",
		},
		{
			name: "instance that is being deleted returns the cleanup plan and uid",
			i: func() *kudoapi.Instance {
				i := idle.DeepCopy()
				i.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Date(2019, 10, 17, 1, 1, 1, 1, time.UTC)}
				i.ObjectMeta.Finalizers = []string{"kudo.dev.instance.cleanup"}
				return i
			}(),
			ov:       ov,
			wantPlan: "cleanup",
			wantUID:  "",
		},
		{
			name: "cleanup is not scheduled again when already running",
			i: func() *kudoapi.Instance {
				i := idle.DeepCopy()
				i.Spec.PlanExecution.PlanName = "cleanup"
				i.Spec.PlanExecution.UID = "111-222-333-444"
				i.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Date(2019, 10, 17, 1, 1, 1, 1, time.UTC)}
				i.ObjectMeta.Finalizers = []string{"kudo.dev.instance.cleanup"}
				return i
			}(),
			ov:       ov,
			wantPlan: "cleanup",
			wantUID:  "111-222-333-444",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			plan, uid := scheduledPlan(tt.i, tt.ov)

			assert.Equal(t, tt.wantPlan, plan, "scheduledPlan() got plan = %v, want %v", plan, tt.wantPlan)
			if tt.wantUID != "" {
				assert.Equal(t, tt.wantUID, uid, "scheduledPlan() got uid = %v, want %v", uid, tt.wantUID)
			}
		})
	}
}

func Test_resetPlanStatusIfPlanIsNew(t *testing.T) {
	status := kudoapi.PlanStatus{
		Name:   "deploy",
		Status: kudoapi.ExecutionInProgress,
		UID:    "111-222-333",
	}

	scheduled := &kudoapi.Instance{
		TypeMeta:   metav1.TypeMeta{APIVersion: "kudo.dev/v1beta1", Kind: "Instance"},
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
		Spec: kudoapi.InstanceSpec{
			PlanExecution: kudoapi.PlanExecution{
				PlanName: "",
				UID:      "",
			},
		},
		Status: kudoapi.InstanceStatus{
			PlanStatus: map[string]kudoapi.PlanStatus{
				"deploy": status,
			},
		},
	}

	tests := []struct {
		name    string
		i       *kudoapi.Instance
		plan    string
		uid     types.UID
		want    *kudoapi.PlanStatus
		wantErr bool
	}{
		{
			name:    "a non-existing plan returns an error",
			i:       scheduled,
			plan:    "fake",
			uid:     "fake-uid",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "an already running plan status is NOT reset",
			i:       scheduled,
			plan:    "deploy",
			uid:     "111-222-333",
			want:    status.DeepCopy(),
			wantErr: false,
		},
		{
			name: "a plan status for a new plan IS reset",
			i:    scheduled,
			plan: "deploy",
			uid:  "222-333-444",
			want: &kudoapi.PlanStatus{
				Name:   "deploy",
				Status: kudoapi.ExecutionPending,
				UID:    "222-333-444",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := resetPlanStatusIfPlanIsNew(tt.i, tt.plan, tt.uid)

			assert.True(t, (err != nil) == tt.wantErr, "resetPlanStatusIfPlanIsNew() error = %v, wantErr %v", err, tt.wantErr)

			if got != nil {
				assert.Equal(t, tt.want.Name, got.Name, "resetPlanStatusIfPlanIsNew() got plan = %v, want %v", got.Name, tt.want.Name)
				assert.Equal(t, tt.want.Status, got.Status, "resetPlanStatusIfPlanIsNew() got status = %v, want %v", got.Status, tt.want.Status)
				assert.Equal(t, tt.want.UID, got.UID, "resetPlanStatusIfPlanIsNew() got uid = %v, want %v", got.UID, tt.want.UID)
			}
		})
	}
}

func Test_ensurePlanStatusInitialized(t *testing.T) {

	makeStatus := func(planName string, status kudoapi.ExecutionStatus, uid types.UID) kudoapi.PlanStatus {
		return kudoapi.PlanStatus{
			Name:   planName,
			Status: status,
			UID:    uid,
			Phases: []kudoapi.PhaseStatus{
				{
					Name:   "phase",
					Status: status,
					Steps: []kudoapi.StepStatus{
						{
							Name:   "step",
							Status: status,
						},
					},
				},
			},
		}
	}

	deployStatus := makeStatus("deploy", kudoapi.ExecutionNeverRun, "")
	oldStatus := makeStatus("old", kudoapi.ExecutionComplete, "222-333-444")
	backupStatus := makeStatus("backup", kudoapi.ExecutionNeverRun, "")
	backupCompleteStatus := makeStatus("backup", kudoapi.ExecutionComplete, "111-222-333")

	instance := &kudoapi.Instance{
		TypeMeta:   metav1.TypeMeta{APIVersion: "kudo.dev/v1beta1", Kind: "Instance"},
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
		Spec:       kudoapi.InstanceSpec{},
		Status:     kudoapi.InstanceStatus{},
	}

	ov := &kudoapi.OperatorVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-operator", Namespace: "default"},
		TypeMeta:   metav1.TypeMeta{Kind: "OperatorVersion", APIVersion: "kudo.dev/v1beta1"},
		Spec: kudoapi.OperatorVersionSpec{
			Plans: map[string]kudoapi.Plan{
				"deploy": {
					Phases: []kudoapi.Phase{
						{
							Name: "phase",
							Steps: []kudoapi.Step{
								{
									Name:  "step",
									Tasks: []string{},
								},
							},
						},
					},
				},
				"backup": {
					Phases: []kudoapi.Phase{
						{
							Name: "phase",
							Steps: []kudoapi.Step{
								{
									Name:  "step",
									Tasks: []string{},
								},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name string
		i    *kudoapi.Instance
		ov   *kudoapi.OperatorVersion
		want kudoapi.InstanceStatus
	}{
		{
			name: "missing plan status IS updated",
			i:    instance.DeepCopy(),
			ov:   ov,
			want: kudoapi.InstanceStatus{
				PlanStatus: map[string]kudoapi.PlanStatus{
					"deploy": deployStatus,
					"backup": backupStatus,
				},
			},
		},
		{
			name: "an existing plan status is NOT updated",
			i: func() *kudoapi.Instance {
				i := instance.DeepCopy()
				i.Status = kudoapi.InstanceStatus{
					PlanStatus: map[string]kudoapi.PlanStatus{"backup": backupCompleteStatus},
				}
				return i
			}(),
			ov: ov,
			want: kudoapi.InstanceStatus{
				PlanStatus: map[string]kudoapi.PlanStatus{
					"deploy": deployStatus,
					"backup": backupCompleteStatus,
				},
			},
		},
		{
			name: "an existing but outdated (missing in OV) plan status is NOT modified",
			i: func() *kudoapi.Instance {
				i := instance.DeepCopy()
				i.Status = kudoapi.InstanceStatus{
					PlanStatus: map[string]kudoapi.PlanStatus{"old": oldStatus},
				}
				return i
			}(),
			ov: ov,
			want: kudoapi.InstanceStatus{
				PlanStatus: map[string]kudoapi.PlanStatus{
					"old":    oldStatus,
					"deploy": deployStatus,
					"backup": backupStatus,
				},
			},
		},
		{
			name: "a new phase is correctly initialized",
			i:    instance.DeepCopy(),
			ov: func() *kudoapi.OperatorVersion {
				modifiedOv := ov.DeepCopy()
				pln := modifiedOv.Spec.Plans["deploy"]
				pln.Phases = append(pln.Phases, kudoapi.Phase{
					Name: "newphase",
					Steps: []kudoapi.Step{
						{
							Name:  "additionalstep",
							Tasks: []string{},
						},
					},
				})
				modifiedOv.Spec.Plans["deploy"] = pln
				return modifiedOv
			}(),
			want: kudoapi.InstanceStatus{
				PlanStatus: map[string]kudoapi.PlanStatus{
					"deploy": {
						Name:   "deploy",
						Status: kudoapi.ExecutionNeverRun,
						UID:    "",
						Phases: []kudoapi.PhaseStatus{
							{
								Name:   "phase",
								Status: kudoapi.ExecutionNeverRun,
								Steps: []kudoapi.StepStatus{
									{
										Name:   "step",
										Status: kudoapi.ExecutionNeverRun,
									},
								},
							},
							{
								Name:   "newphase",
								Status: kudoapi.ExecutionNeverRun,
								Steps: []kudoapi.StepStatus{
									{
										Name:   "additionalstep",
										Status: kudoapi.ExecutionNeverRun,
									},
								},
							},
						},
					},
					"backup": backupStatus,
				},
			},
		},
		{
			name: "a new step is correctly initialized",
			i:    instance.DeepCopy(),
			ov: func() *kudoapi.OperatorVersion {
				modifiedOv := ov.DeepCopy()
				pln := modifiedOv.Spec.Plans["deploy"]
				pln.Phases[0].Steps = append([]kudoapi.Step{
					{
						Name:  "additionalstep",
						Tasks: []string{},
					},
				}, pln.Phases[0].Steps...)
				modifiedOv.Spec.Plans["deploy"] = pln
				return modifiedOv
			}(),
			want: kudoapi.InstanceStatus{
				PlanStatus: map[string]kudoapi.PlanStatus{
					"deploy": {
						Name:   "deploy",
						Status: kudoapi.ExecutionNeverRun,
						UID:    "",
						Phases: []kudoapi.PhaseStatus{
							{
								Name:   "phase",
								Status: kudoapi.ExecutionNeverRun,
								Steps: []kudoapi.StepStatus{
									{
										Name:   "additionalstep",
										Status: kudoapi.ExecutionNeverRun,
									},
									{
										Name:   "step",
										Status: kudoapi.ExecutionNeverRun,
									},
								},
							},
						},
					},
					"backup": backupStatus,
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ensurePlanStatusInitialized(tt.i, tt.ov)

			assert.Equal(t, tt.want, tt.i.Status)

			fmt.Printf("\n==== %s ====\n", tt.name)
			s, _ := json.MarshalIndent(tt.i, "", "  ")
			fmt.Println(string(s))
		})
	}
}

func Test_retryReconciliation(t *testing.T) {
	instance := &kudoapi.Instance{
		TypeMeta:   metav1.TypeMeta{APIVersion: "kudo.dev/v1beta1", Kind: "Instance"},
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
		Spec: kudoapi.InstanceSpec{
			PlanExecution: kudoapi.PlanExecution{
				Status: kudoapi.ExecutionComplete,
			},
		},
		Status: kudoapi.InstanceStatus{},
	}
	timeNow := time.Now()
	deployPlanName := "deploy"

	tests := []struct {
		name string
		i    *kudoapi.Instance
		want reconcile.Result
	}{
		{"finished plan", instance, reconcile.Result{}},
		{"just started plan", func() *kudoapi.Instance {
			i := instance.DeepCopy()
			i.Spec.PlanExecution.Status = kudoapi.ExecutionInProgress
			i.Spec.PlanExecution.PlanName = deployPlanName
			i.Status.PlanStatus = map[string]kudoapi.PlanStatus{
				deployPlanName: {LastUpdatedTimestamp: &metav1.Time{Time: timeNow}},
			}
			return i
		}(), reconcile.Result{Requeue: true, RequeueAfter: 1 * time.Second}},
		{"2 minutes old update", func() *kudoapi.Instance {
			i := instance.DeepCopy()
			i.Spec.PlanExecution.Status = kudoapi.ExecutionInProgress
			i.Spec.PlanExecution.PlanName = deployPlanName
			i.Status.PlanStatus = map[string]kudoapi.PlanStatus{
				deployPlanName: {LastUpdatedTimestamp: &metav1.Time{Time: timeNow.Add(-2 * time.Minute)}},
			}
			return i
		}(), reconcile.Result{Requeue: true, RequeueAfter: 3 * time.Second}},
		{"long stalled plan", func() *kudoapi.Instance {
			i := instance.DeepCopy()
			i.Spec.PlanExecution.Status = kudoapi.ExecutionInProgress
			i.Spec.PlanExecution.PlanName = deployPlanName
			i.Status.PlanStatus = map[string]kudoapi.PlanStatus{
				deployPlanName: {LastUpdatedTimestamp: &metav1.Time{Time: timeNow.Add(-2 * time.Hour)}},
			}
			return i
		}(), reconcile.Result{Requeue: true, RequeueAfter: 60 * time.Second}},
	}
	for _, tt := range tests {
		result := computeTheReconcileResult(tt.i, func() time.Time { return timeNow })
		if result != tt.want {
			t.Errorf("%s: expected %v but got %v", tt.name, tt.want, result)
		}
	}
}
