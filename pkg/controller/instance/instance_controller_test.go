package instance

import (
	"context"
	"log"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var c client.Client
var expectedRequest = reconcile.Request{NamespacedName: types.NamespacedName{Name: "deploy", Namespace: "default"}}

const timeout = time.Second * 5

func TestReconcile_InstancesOnFrameworkVersionEvent(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	// Setup the Manager and Controller. Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	c = mgr.GetClient()

	// Given an existing Instance object
	log.Printf("Given an existing instance \"foo-instance\"")
	instance := &v1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-instance", Namespace: "default", Labels: map[string]string{"framework": "foo-framework"}},
		Spec: v1alpha1.InstanceSpec{
			FrameworkVersion: v1.ObjectReference{
				Name:      "foo-framework",
				Namespace: "default",
			},
		},
		Status: v1alpha1.InstanceStatus{
			ActivePlan: v1.ObjectReference{
				Name:      "deploy",
				Namespace: "default",
			},
		},
	}
	assert.NoError(t, c.Create(context.TODO(), instance))
	defer c.Delete(context.TODO(), instance)

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
	frameworkVersion := &v1alpha1.FrameworkVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-framework", Namespace: "default"},
		Spec: v1alpha1.FrameworkVersionSpec{
			Plans: map[string]v1alpha1.Plan{"deploy": {}},
		},
	}
	assert.NoError(t, c.Create(context.TODO(), frameworkVersion))

	defer c.Delete(context.TODO(), frameworkVersion)

	log.Printf("Then its instances are reconciled")
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expectedRequest)))
}
