package instance

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/onsi/ginkgo"

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

	stopMgr, mgrStopped, c := startTestManager(g)

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
	gomega.Eventually(func() bool {
		peList := &v1alpha1.PlanExecutionList{}
		err := c.List(
			context.TODO(),
			peList,
			client.MatchingLabels(map[string]string{
				kudo.InstanceLabel: in.Name,
			}))
		assert.NoError(t, err)
		return len(peList.Items) == 1 && strings.Contains(peList.Items[0].Name, "foo-instance-deploy")
	}, timeout).Should(gomega.BeTrue())

	log.Print("When we stop the manager")
	close(stopMgr)
	mgrStopped.Wait()

	log.Print("And update the instance parameter value")
	in.Spec.Parameters = map[string]string{ "param": "newvalue" }
	assert.NoError(t, c.Update(context.TODO(), in))

	log.Print("And restart the manager again")
	stopMgr, mgrStopped, c = startTestManager(g)

	log.Print("Then an update plan should be triggered instead of deploy plan")
	gomega.Eventually(func() bool {
		peList := &v1alpha1.PlanExecutionList{}
		err := c.List(
			context.TODO(),
			peList,
			client.MatchingLabels(map[string]string{
				kudo.InstanceLabel: in.Name,
			}))
		assert.NoError(t, err)
		for _, item := range peList.Items {
			fmt.Println(item.Name)
			if strings.Contains(item.Name, "foo-instance-update") {
				return true
			}
		}
		return false
	}, timeout).Should(gomega.BeTrue())

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()
}

func startTestManager(g *gomega.GomegaWithT) (chan struct{}, *sync.WaitGroup, client.Client) {
	mgr, err := manager.New(cfg, manager.Options{})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	c := mgr.GetClient()
	reconciler := newReconciler(mgr)
	g.Expect(add(mgr, reconciler)).NotTo(gomega.HaveOccurred())

	stop := make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		g.Expect(mgr.Start(stop)).NotTo(gomega.HaveOccurred())
		wg.Done()
	}()
	return stop, wg, c
}
