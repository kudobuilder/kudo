// +build integration

package webhook

import (
	"context"
	"fmt"
	"log"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/kudobuilder/kudo/pkg/apis"
	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/crd"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/prereq"
)

func TestSource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "Webhook Test Suite", []Reporter{printer.NewlineReporter{}})
}

var env *envtest.Environment
var instanceAdmissionWebhookPath string

var _ = BeforeSuite(func() {
	log.SetOutput(GinkgoWriter)
	env = &envtest.Environment{}

	// 1. add webhook configuration to the env. we use the same webhook configuration that `kudo init` generates
	log.Print("test.BeforeSuite: initializing webhook configuration")
	iaw := prereq.InstanceAdmissionWebhook(v1.NamespaceDefault)
	instanceAdmissionWebhookPath = *iaw.Webhooks[0].ClientConfig.Service.Path

	env.WebhookInstallOptions = envtest.WebhookInstallOptions{MutatingWebhooks: []runtime.Object{&iaw}}

	// 2. add KUDO CRDs
	log.Print("test.BeforeSuite: initializing CRDs")
	env.CRDs = crd.NewInitializer().Resources()

	_, err := env.Start()
	Expect(err).NotTo(HaveOccurred())
}, envtest.StartTimeout)

var _ = AfterSuite(func() {
	Expect(env.Stop()).NotTo(HaveOccurred())
}, envtest.StopTimeout)

var _ = Describe("Test", func() {

	var defaultTimeout float64 = 5 // seconds

	var c client.Client
	var stop chan struct{}

	// every test case gets it's own manager and client instances. not sure if that's the best
	// practice but it  seems to be fast enough.
	BeforeEach(func() {
		// 1. initializing manager
		log.Print("test.BeforeEach: initializing manager")
		mgr, err := manager.New(env.Config, manager.Options{
			Port:               env.WebhookInstallOptions.LocalServingPort,
			Host:               env.WebhookInstallOptions.LocalServingHost,
			CertDir:            env.WebhookInstallOptions.LocalServingCertDir,
			MetricsBindAddress: "0", // disable metrics server
		})
		Expect(err).NotTo(HaveOccurred())

		// 2. initializing scheme with v1beta1 types
		log.Print("test.BeforeEach: initializing scheme")
		err = apis.AddToScheme(mgr.GetScheme())
		Expect(err).NotTo(HaveOccurred())

		// 3. creating the client. **Note:** client.New method will create an uncached client, a cached one
		// (e.g. mgr.GetClient) leads to caching issues in this test.
		log.Print("test.BeforeEach: initializing client")
		c, err = client.New(env.Config, client.Options{Scheme: mgr.GetScheme()})
		Expect(err).NotTo(HaveOccurred())

		// 4. registering instance admission controller
		log.Print("test.BeforeEach: initializing webhook server")
		server := mgr.GetWebhookServer()
		server.Register(instanceAdmissionWebhookPath, &ctrhook.Admission{Handler: &InstanceAdmission{client: c}})

		// 5. starting the manager
		stop = make(chan struct{})
		go func() {
			err = mgr.Start(stop)
			Expect(err).NotTo(HaveOccurred())
		}()
	})

	AfterEach(func() {
		close(stop)
	})

	deploy := kudoapi.DeployPlanName
	update := kudoapi.UpdatePlanName
	cleanup := kudoapi.CleanupPlanName

	ov := &kudoapi.OperatorVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-operator", Namespace: "default"},
		TypeMeta:   metav1.TypeMeta{Kind: "OperatorVersion", APIVersion: "kudo.dev/v1beta1"},
		Spec: kudoapi.OperatorVersionSpec{
			Plans: map[string]kudoapi.Plan{deploy: {}, update: {}, cleanup: {}},
			Parameters: []kudoapi.Parameter{
				{
					Name:    "foo",
					Trigger: deploy,
				},
				{
					Name:    "bar",
					Trigger: update,
				},
			},
		},
	}

	idle := &kudoapi.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo-instance",
			Namespace: "default",
		},
		Spec: kudoapi.InstanceSpec{
			OperatorVersion: v1.ObjectReference{Name: "foo-operator"},
		},
	}

	Describe("Instance admission controller", func() {
		var key client.ObjectKey

		It("should allow instance creation", func() {
			// 1. create the OV first
			ov := ov.DeepCopy()
			err := c.Create(context.TODO(), ov)
			Expect(err).NotTo(HaveOccurred())

			// 2. create the initial Instance
			err = c.Create(context.TODO(), idle)
			Expect(err).NotTo(HaveOccurred())
		}, defaultTimeout)

		It("should schedule deploy plan when an instance is created", func() {
			// 1. admission controller should schedule 'deploy' plan and add cleanup finalizer for new instances
			key = keyFrom(idle)
			i := instanceWith(c, key)

			Expect(i.Spec.PlanExecution.PlanName).Should(Equal(deploy))
			Expect(i.Spec.PlanExecution.UID).ShouldNot(BeEmpty())
			Expect(i.HasCleanupFinalizer()).Should(BeTrue())
		}, defaultTimeout)

		It("should allow rescheduling of the deploy plan directly", func() {
			i := instanceWith(c, key)

			// 1. restart currently running plan by resetting the UID
			uid := i.Spec.PlanExecution.UID // save current plan UID for later checks
			i.Spec.PlanExecution.UID = ""

			err := c.Update(context.TODO(), i)
			Expect(err).NotTo(HaveOccurred())
			Expect(i.Spec.PlanExecution.PlanName).Should(Equal(deploy))
			Expect(i.Spec.PlanExecution.UID).ShouldNot(BeEmpty())
			Expect(i.Spec.PlanExecution.UID).NotTo(Equal(uid)) // same plan but new UID
		}, defaultTimeout)

		It("should allow rescheduling of the deploy plan through parameter update", func() {
			i := instanceWith(c, key)

			// 1. restart currently running plan by resetting the UID
			uid := i.Spec.PlanExecution.UID // save current plan UID for later checks
			i.Spec.Parameters = map[string]string{"foo": "2"}

			err := c.Update(context.TODO(), i)
			Expect(err).NotTo(HaveOccurred())
			Expect(i.Spec.PlanExecution.PlanName).Should(Equal(deploy))
			Expect(i.Spec.PlanExecution.UID).ShouldNot(BeEmpty())
			Expect(i.Spec.PlanExecution.UID).NotTo(Equal(uid)) // same plan but new UID
			Expect(i.Spec.PlanExecution.Status).Should(Equal(kudoapi.ExecutionNeverRun))
		}, defaultTimeout)

		It("should NOT allow scheduling of another plan while deploy is running", func() {
			i := instanceWith(c, key)

			// 1. try to start the 'update' plan while another plan is in progress
			i.Spec.PlanExecution.PlanName = update

			err := c.Update(context.TODO(), i)
			Expect(err).To(HaveOccurred()) // boom
		}, defaultTimeout)

		It("should NOT allow upgrading instance while a plan is running", func() {
			// 1. create new OV first
			newOv := ov.DeepCopy()
			newOv.Name = fmt.Sprintf("%s-%s", ov.Name, "2.0")
			err := c.Create(context.TODO(), newOv)
			Expect(err).NotTo(HaveOccurred())

			i := instanceWith(c, key)

			// 2. bump instance OV reference
			i.Spec.OperatorVersion.Name = newOv.Name
			err = c.Update(context.TODO(), i)
			Expect(err).To(HaveOccurred()) // boom
		}, defaultTimeout)

		It("should NOT allow updating parameters while another plan is running", func() {
			i := instanceWith(c, key)

			// 1. try to instance parameters that would trigger a different plan while another plan is in progress
			i.Spec.Parameters = map[string]string{"bar": "2"}

			err := c.Update(context.TODO(), i)
			Expect(err).To(HaveOccurred()) // boom
		}, defaultTimeout)

		It("should clear scheduled plan when it is completed", func() {
			i := instanceWith(c, key)

			// 1. finish the 'deploy' plan by updating the status field
			i.Spec.PlanExecution.Status = kudoapi.ExecutionComplete
			err := c.Update(context.TODO(), i)
			Expect(err).NotTo(HaveOccurred())

			// 2. admission controller should reset the Spec.PlanExecution
			i = instanceWith(c, key)
			Expect(string(i.Spec.PlanExecution.Status)).Should(Equal(""))
			Expect(string(i.Spec.PlanExecution.UID)).Should(Equal(""))
			Expect(i.Spec.PlanExecution.PlanName).Should(Equal(""))

		}, defaultTimeout)

		It("cleanup plan can NOT be scheduled if the instance is not being deleted", func() {
			i := instanceWith(c, key)

			// 1. finish the 'deploy' plan by updating the status field
			i.Spec.PlanExecution.PlanName = kudoapi.CleanupPlanName
			err := c.Update(context.TODO(), i)
			Expect(err).To(HaveOccurred())

		}, defaultTimeout)

		It("cleanup plan CAN be scheduled if the instance is being deleted", func() {
			// 1. delete the instance
			i := instanceWith(c, key)
			err := c.Delete(context.TODO(), i)
			Expect(err).NotTo(HaveOccurred())

			// 2. schedule the 'cleanup' plan like the controller would
			i = instanceWith(c, key)
			uid := i.Spec.PlanExecution.UID

			i.Spec.PlanExecution.PlanName = kudoapi.CleanupPlanName
			i.Spec.PlanExecution.UID = uuid.NewUUID()
			err = c.Update(context.TODO(), i)
			Expect(err).NotTo(HaveOccurred())

			// 3. admission controller should reset the Spec.PlanExecution
			i = instanceWith(c, key)
			Expect(i.Spec.PlanExecution.PlanName).Should(Equal(kudoapi.CleanupPlanName))
			Expect(i.Spec.PlanExecution.UID).ShouldNot(Equal(uid))

		}, defaultTimeout)

		It("another plan can NOT be scheduled while cleanup is running", func() {
			i := instanceWith(c, key)

			// 1. finish the 'deploy' plan by updating the status field
			i.Spec.PlanExecution.PlanName = kudoapi.DeployPlanName
			err := c.Update(context.TODO(), i)
			Expect(err).To(HaveOccurred())

		}, defaultTimeout)
	})
})

// keyFrom method is a small helper method to retrieve an ObjectKey from the given object. Meant to be used within this test class
// only as it will fail the test should an error occur.
func keyFrom(obj runtime.Object) client.ObjectKey {
	key, err := client.ObjectKeyFromObject(obj)
	Expect(err).NotTo(HaveOccurred())
	return key
}

// instanceWith method is a small helper method to retrieve an Instance with the give key. Meant to be used within this test class
// only as it will fail the test should an error occur.
func instanceWith(c client.Client, key client.ObjectKey) *kudoapi.Instance {
	i := &kudoapi.Instance{}
	err := c.Get(context.TODO(), key, i)
	Expect(err).NotTo(HaveOccurred())
	return i
}
