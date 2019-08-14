package instance

import (
	"context"
	"github.com/onsi/ginkgo"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/util/kudo"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const timeout = time.Second * 5

func TestRestartController(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	gomega.RegisterFailHandler(ginkgo.Fail)

	// Setup the Manager and Controller. Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	c := mgr.GetClient()

	log.Printf("Given an existing instance 'foo-instance' and operator 'foo-operator'")
	in := &v1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-instance", Namespace: "default", Labels: map[string]string{"framework": "foo-framework"}},
		Spec: v1alpha1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name:      "foo-operator",
				Namespace: "default",
			},
			Parameters: map[string]string{"param": "value"},
		},
	}
	assert.NoError(t, c.Create(context.TODO(), in))
	defer c.Delete(context.TODO(), in)

	ov := &v1alpha1.OperatorVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-operator", Namespace: "default"},
		TypeMeta:   metav1.TypeMeta{Kind: "OperatorVersion", APIVersion: "kudo.k8s.io/v1alpha1"},
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

	_ = newReconciler(mgr)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	stopMgr, mgrStopped := startTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	log.Print("And a deploy plan that was already run")
	assert.NoError(t, err)
	gomega.Eventually(func() bool {
		peList := &v1alpha1.PlanExecutionList{}
		err = c.List(
			context.TODO(),
			peList,
			client.MatchingLabels(map[string]string{
				kudo.OperatorLabel: ov.Name,
				kudo.InstanceLabel: in.Name,
			}))
		return len(peList.Items) > 0 && strings.Contains(peList.Items[0].Name, "foo-instance-deploy")
	}, timeout).Should(gomega.BeTrue())

	/*log.Print("When we stop the manager")
	close(stopMgr)
	mgrStopped.Wait()

	log.Print("And update the instance parameter value")
	in.Spec.Parameters = map[string]string{ "param": "newvalue" }

	stopMgr, mgrStopped = startTestManager(mgr, g)
	assert.NoError(t, c.Update(context.TODO(), in))

	log.Print("And restart the manager again")
	stopMgr, mgrStopped = startTestManager(mgr, g)

	log.Print("Then an update plan should be triggered instead of deploy plan")
	peList = &v1alpha1.PlanExecutionList{}
	err = c.List(
		context.TODO(),
		peList,
		client.MatchingLabels(map[string]string{
			kudo.OperatorLabel: ov.Name,
			kudo.InstanceLabel: in.Name,
		}))
	assert.NoError(t, err)
	assert.True(t, strings.Contains(peList.Items[0].Name, "foo-instance-update"))

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()*/
}

func startTestManager(mgr manager.Manager, g *gomega.GomegaWithT) (chan struct{}, *sync.WaitGroup) {
	stop := make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		g.Expect(mgr.Start(stop)).NotTo(gomega.HaveOccurred())
		wg.Done()
	}()
	return stop, wg
}
