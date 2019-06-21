package instance

import (
	"context"
	"log"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const timeout = time.Second * 5

func TestReconcile_InstancesOnFrameworkVersionEvent(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Setup the Manager and Controller. Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	c := mgr.GetClient()

	// Given an existing Instance object
	log.Printf("Given an existing instance \"foo-instance\"")
	in := &v1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-instance", Namespace: "default", Labels: map[string]string{"framework": "foo-framework"}},
		Spec: v1alpha1.InstanceSpec{
			FrameworkVersion: v1.ObjectReference{
				Name:      "foo-framework",
				Namespace: "default",
			},
		},
	}
	assert.NoError(t, c.Create(context.TODO(), in))
	defer c.Delete(context.TODO(), in)

	reconciler := newReconciler(mgr)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	recFn, requests := SetupTestReconcile(reconciler)
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// Create a FrameworkVersion object with an empty "deploy" plan first
	log.Printf("When a frameworkVersion is created")
	fv := &v1alpha1.FrameworkVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-framework", Namespace: "default"},
		TypeMeta:   metav1.TypeMeta{Kind: "FrameworkVersion", APIVersion: "kudo.k8s.io/v1alpha1"},
		Spec: v1alpha1.FrameworkVersionSpec{
			Plans: map[string]v1alpha1.Plan{"deploy": {}},
		},
	}
	assert.NoError(t, c.Create(context.TODO(), fv))

	defer c.Delete(context.TODO(), fv)

	log.Printf("Then its instances are reconciled")
	var key, _ = client.ObjectKeyFromObject(in)
	var expected = reconcile.Request{NamespacedName: key}
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expected)))

	log.Printf("And a default deploy execution plan was created")
	peList := &v1alpha1.PlanExecutionList{}
	err = c.List(
		context.TODO(),
		client.MatchingLabels(map[string]string{
			"framework-version": fv.Name,
			"instance":          in.Name,
		}),
		peList)
	assert.NoError(t, err)
	assert.True(t, strings.Contains(peList.Items[0].Name, "foo-instance-deploy"))
}
