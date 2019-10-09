package instance

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/util/kudo"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const timeout = time.Second * 5
const tick = time.Millisecond * 500

func TestRestartController(t *testing.T) {
	stopMgr, mgrStopped, c := startTestManager(t)

	log.Printf("Given an existing instance 'foo-instance' and operator 'foo-operator'")
	in := &v1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-instance", Namespace: "default", Labels: map[string]string{kudo.OperatorLabel: "foo-operator"}},
		Spec: v1alpha1.InstanceSpec{
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

	ov := &v1alpha1.OperatorVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-operator", Namespace: "default"},
		TypeMeta:   metav1.TypeMeta{Kind: "OperatorVersion", APIVersion: "kudo.dev/v1alpha1"},
		Spec: v1alpha1.OperatorVersionSpec{
			Plans: map[string]v1alpha1.Plan{"deploy": {}, "update": {}},
			Parameters: []v1alpha1.Parameter{
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
	i := &v1alpha1.Instance{}
	err := c.Get(context.TODO(), key, i)
	if err != nil {
		fmt.Printf("%w", err)
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
