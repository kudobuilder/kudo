package instance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
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
		plan     *v1beta1.Plan
		tasks    []v1beta1.Task
		emeta    *engine.Metadata
		want     map[string]string
		wantErr  bool
	}{
		{
			name:     "no tasks, no pipes",
			planName: "deploy",
			plan: &v1beta1.Plan{Strategy: "serial", Phases: []v1beta1.Phase{
				{
					Name: "phase", Strategy: "serial", Steps: []v1beta1.Step{
						{
							Name: "step", Tasks: []string{}},
					}},
			}},
			tasks: []v1beta1.Task{},
			emeta: meta,
			want:  map[string]string{},
		},
		{
			name:     "no pipe tasks, no pipes",
			planName: "deploy",
			plan: &v1beta1.Plan{Strategy: "serial", Phases: []v1beta1.Phase{
				{
					Name: "phase", Strategy: "serial", Steps: []v1beta1.Step{
						{
							Name: "step", Tasks: []string{"task"}},
					}},
			}},
			tasks: []v1beta1.Task{
				{
					Name: "task",
					Kind: "Dummy",
					Spec: v1beta1.TaskSpec{
						DummyTaskSpec: v1beta1.DummyTaskSpec{Done: false},
					},
				},
			},
			emeta: meta,
			want:  map[string]string{},
		},
		{
			name:     "one pipe task, one pipes element",
			planName: "deploy",
			plan: &v1beta1.Plan{Strategy: "serial", Phases: []v1beta1.Phase{
				{
					Name: "phase", Strategy: "serial", Steps: []v1beta1.Step{
						{
							Name: "step", Tasks: []string{"task"}},
					}},
			}},
			tasks: []v1beta1.Task{
				{
					Name: "task",
					Kind: "Pipe",
					Spec: v1beta1.TaskSpec{
						PipeTaskSpec: v1beta1.PipeTaskSpec{
							Pod: "pipe-pod.yaml",
							Pipe: []v1beta1.PipeSpec{
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
			plan: &v1beta1.Plan{Strategy: "serial", Phases: []v1beta1.Phase{
				{
					Name: "phase", Strategy: "serial", Steps: []v1beta1.Step{
						{Name: "stepOne", Tasks: []string{"task-one"}},
						{Name: "stepTwo", Tasks: []string{"task-two"}},
					}},
			}},
			tasks: []v1beta1.Task{
				{
					Name: "task-one",
					Kind: "Pipe",
					Spec: v1beta1.TaskSpec{
						PipeTaskSpec: v1beta1.PipeTaskSpec{
							Pod: "pipe-pod.yaml",
							Pipe: []v1beta1.PipeSpec{
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
					Spec: v1beta1.TaskSpec{
						PipeTaskSpec: v1beta1.PipeTaskSpec{
							Pod: "pipe-pod.yaml",
							Pipe: []v1beta1.PipeSpec{
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
			plan: &v1beta1.Plan{Strategy: "serial", Phases: []v1beta1.Phase{
				{
					Name: "phase", Strategy: "serial", Steps: []v1beta1.Step{
						{
							Name: "step", Tasks: []string{"task"}},
					}},
			}},
			tasks: []v1beta1.Task{
				{
					Name: "task",
					Kind: "Pipe",
					Spec: v1beta1.TaskSpec{
						PipeTaskSpec: v1beta1.PipeTaskSpec{
							Pod: "pipe-pod.yaml",
							Pipe: []v1beta1.PipeSpec{
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
			diff := v1beta1.ParameterDiff(old, tt.new)
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
			changed, removed := v1beta1.RichParameterDiff(old, tt.new)
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
	ov := &v1beta1.OperatorVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-operator", Namespace: "default"},
		TypeMeta:   metav1.TypeMeta{Kind: "OperatorVersion", APIVersion: "kudo.dev/v1beta1"},
		Spec:       v1beta1.OperatorVersionSpec{Plans: map[string]v1beta1.Plan{"cleanup": {}}},
	}

	idle := &v1beta1.Instance{
		TypeMeta:   metav1.TypeMeta{APIVersion: "kudo.dev/v1beta1", Kind: "Instance"},
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
		Spec: v1beta1.InstanceSpec{
			PlanExecution: v1beta1.PlanExecution{
				PlanName: "",
				UID:      "",
			},
		},
	}

	tests := []struct {
		name     string
		i        *v1beta1.Instance
		ov       *v1beta1.OperatorVersion
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
			i: func() *v1beta1.Instance {
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
			i: func() *v1beta1.Instance {
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
			i: func() *v1beta1.Instance {
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

func Test_resetPlanStatusIfThePlanIsNew(t *testing.T) {
	status := v1beta1.PlanStatus{
		Name:   "deploy",
		Status: v1beta1.ExecutionInProgress,
		UID:    "111-222-333",
	}

	scheduled := &v1beta1.Instance{
		TypeMeta:   metav1.TypeMeta{APIVersion: "kudo.dev/v1beta1", Kind: "Instance"},
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "test"},
		Spec: v1beta1.InstanceSpec{
			PlanExecution: v1beta1.PlanExecution{
				PlanName: "",
				UID:      "",
			},
		},
		Status: v1beta1.InstanceStatus{
			PlanStatus: map[string]v1beta1.PlanStatus{
				"deploy": status,
			},
		},
	}

	tests := []struct {
		name    string
		i       *v1beta1.Instance
		plan    string
		uid     types.UID
		want    *v1beta1.PlanStatus
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
			want: &v1beta1.PlanStatus{
				Name:   "deploy",
				Status: v1beta1.ExecutionPending,
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
