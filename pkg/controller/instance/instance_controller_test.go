// +build integration

package instance

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/kudobuilder/kudo/pkg/apis"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

const timeout = time.Second * 5
const tick = time.Millisecond * 500

var cfg *rest.Config

func TestMain(m *testing.M) {
	t := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "config", "crds")},
	}

	if err := apis.AddToScheme(scheme.Scheme); err != nil {
		log.Fatal(err)
	}

	var err error
	if cfg, err = t.Start(); err != nil {
		log.Fatal(err)
	}

	code := m.Run()
	// t.Stop() returns an error, but since we exit in the next line anyway, it is suppressed
	t.Stop() //nolint:errcheck
	os.Exit(code)
}

func TestRestartController(t *testing.T) {
	stopMgr, mgrStopped, c := startTestManager(t)

	log.Printf("Given an existing instance 'foo-instance' and operator 'foo-operator'")
	in := &v1beta1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-instance", Namespace: "default", Labels: map[string]string{kudo.OperatorLabel: "foo-operator"}},
		Spec: v1beta1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name:      "foo-operator",
				Namespace: "default",
			},
			Parameters: map[string]string{"param": "value"},
		},
	}
	instanceKey, _ := client.ObjectKeyFromObject(in)
	assert.NoError(t, c.Create(context.TODO(), in))
	defer c.Delete(context.TODO(), in)

	ov := &v1beta1.OperatorVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-operator", Namespace: "default"},
		TypeMeta:   metav1.TypeMeta{Kind: "OperatorVersion", APIVersion: "kudo.dev/v1beta1"},
		Spec: v1beta1.OperatorVersionSpec{
			Plans: map[string]v1beta1.Plan{"deploy": {}, "update": {}},
			Parameters: []v1beta1.Parameter{
				{
					Name:    "param",
					Default: kudo.String("default"),
				},
			},
		},
	}
	assert.NoError(t, c.Create(context.TODO(), ov))
	defer c.Delete(context.TODO(), ov)

	log.Print("And a deploy plan that was already run")
	assert.Eventually(t, func() bool { return instancePlanFinished(instanceKey, "deploy", c) }, timeout, tick)

	log.Print("When we stop the manager")
	close(stopMgr)
	mgrStopped.Wait()

	log.Print("And update the instance parameter value")
	err := c.Get(context.TODO(), instanceKey, in) // we need to pull here again because the state of instance was modified in between
	assert.NoError(t, err)
	in.Spec.Parameters = map[string]string{"param": "newvalue"}
	assert.NoError(t, c.Update(context.TODO(), in))

	log.Print("And restart the manager again")
	stopMgr, mgrStopped, c = startTestManager(t)

	log.Print("Then an update plan should be triggered instead of deploy plan")
	assert.Eventually(t, func() bool { return instancePlanFinished(instanceKey, "update", c) }, timeout, tick)

	close(stopMgr)
	mgrStopped.Wait()
}

func instancePlanFinished(key client.ObjectKey, planName string, c client.Client) bool {
	i := &v1beta1.Instance{}
	err := c.Get(context.TODO(), key, i)
	if err != nil {
		fmt.Printf("%v", err)
		return false
	}
	return i.Status.PlanStatus[planName].Status.IsFinished()
}

func startTestManager(t *testing.T) (chan struct{}, *sync.WaitGroup, client.Client) {
	mgr, err := manager.New(cfg, manager.Options{})
	assert.Nil(t, err, "Error when creating manager")
	err = (&Reconciler{
		Client:   mgr.GetClient(),
		Recorder: mgr.GetEventRecorderFor("instance-controller"),
		Scheme:   mgr.GetScheme(),
	}).SetupWithManager(mgr)

	stop := make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		_ = mgr.Start(stop)
		wg.Done()
	}()
	return stop, wg, mgr.GetClient()
}

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
			got, err := pipesMap(tt.planName, tt.plan, tt.tasks, tt.emeta)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("pipesMap() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
