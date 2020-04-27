package instance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
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

func Test_fetchNewExecutionPlan(t *testing.T) {
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
	if err := idle.AnnotateSnapshot(); err != nil {
		t.Fatalf("failed to annotate instance snaphot: %v", err)
	}

	deploy := "deploy"
	cleanup := "cleanup"
	backup := "backup"
	testUUID := uuid.NewUUID()

	deploying := idle.DeepCopy()
	deploying.Spec.PlanExecution = v1beta1.PlanExecution{PlanName: deploy, UID: testUUID}
	if err := deploying.AnnotateSnapshot(); err != nil {
		t.Fatalf("failed to annotate instance snaphot: %v", err)
	}

	deleted := deploying.DeepCopy()
	deleted.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Date(2019, 10, 17, 1, 1, 1, 1, time.UTC)}
	deleted.ObjectMeta.Finalizers = []string{"kudo.dev.instance.cleanup"}

	uninstalling := deleted.DeepCopy()
	uninstalling.Spec.PlanExecution.PlanName = cleanup
	if err := uninstalling.AnnotateSnapshot(); err != nil {
		t.Fatalf("failed to annotate instance snaphot: %v", err)
	}

	tests := []struct {
		name    string
		i       *v1beta1.Instance
		ov      *v1beta1.OperatorVersion
		want    *string
		wantErr bool
	}{
		{
			name:    "no change means no new plan",
			i:       idle,
			ov:      ov,
			want:    nil,
			wantErr: false,
		},
		{
			name: "deploy plan is scheduled without new UID",
			i: func() *v1beta1.Instance {
				i := idle.DeepCopy()
				i.Spec.PlanExecution.PlanName = deploy
				return i
			}(),
			ov:      ov,
			want:    &deploy,
			wantErr: false,
		},
		{
			name: "deploy plan is scheduled WITH new UID",
			i: func() *v1beta1.Instance {
				i := idle.DeepCopy()
				i.Spec.PlanExecution.PlanName = deploy
				i.Spec.PlanExecution.UID = uuid.NewUUID()
				return i
			}(),
			ov:      ov,
			want:    &deploy,
			wantErr: false,
		},
		{
			name: "no new plan was scheduled WITH a new UID",
			i: func() *v1beta1.Instance {
				i := idle.DeepCopy()
				i.Spec.PlanExecution.UID = uuid.NewUUID()
				return i
			}(),
			ov:      ov,
			want:    nil,
			wantErr: false,
		},
		{
			name: "new plan overrides deploy plan",
			i: func() *v1beta1.Instance {
				i := deploying.DeepCopy()
				i.Spec.PlanExecution.PlanName = backup
				return i
			}(),
			ov:      ov,
			want:    &backup,
			wantErr: false,
		},
		{
			name: "new plan overrides deploy plan WITH new UID",
			i: func() *v1beta1.Instance {
				i := deploying.DeepCopy()
				i.Spec.PlanExecution.PlanName = backup
				i.Spec.PlanExecution.UID = uuid.NewUUID()
				return i
			}(),
			ov:      ov,
			want:    &backup,
			wantErr: false,
		},
		{
			name:    "cleanup is scheduled for a deleted instance",
			i:       deleted,
			ov:      ov,
			want:    &cleanup,
			wantErr: false,
		},
		{
			name:    "cleanup is NOT scheduled when it is already scheduled",
			i:       uninstalling,
			ov:      ov,
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fetchNewExecutionPlan(tt.i, tt.ov)
			assert.Equal(t, tt.wantErr, err != nil, "expected error %v, but got %v", tt.wantErr, err)
			assert.Equal(t, tt.want, got, "expected '%s' plan returned but got: '%s'", stringPtrToString(tt.want), stringPtrToString(got))
		})
	}
}

func stringPtrToString(p *string) string {
	if p != nil {
		return *p
	}
	return "<nil>"
}
