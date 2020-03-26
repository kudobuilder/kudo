package webhook

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrhook "sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/kudobuilder/kudo/pkg/apis"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/crd"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/prereq"
	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
)

func TestSource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "Webhook Test Suite", []Reporter{printer.NewlineReporter{}})
}

var env *envtest.Environment
var instanceAdmissionWebhookPath string

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter)))
	env = &envtest.Environment{}

	// 1. add webhook configuration to the env. we use the same webhook configuration that `kudo init` generates
	logf.Log.Info("test.BeforeSuite: initializing webhook configuration")
	iaw := prereq.InstanceAdmissionWebhook(v1.NamespaceDefault)
	instanceAdmissionWebhookPath = *iaw.Webhooks[0].ClientConfig.Service.Path

	env.WebhookInstallOptions = envtest.WebhookInstallOptions{MutatingWebhooks: []runtime.Object{&iaw}}

	// 2. add KUDO CRDs
	logf.Log.Info("test.BeforeSuite: initializing CRDs")
	env.CRDs = crd.NewInitializer().Resources()

	_, err := env.Start()
	Expect(err).NotTo(HaveOccurred())

	close(done)
}, envtest.StartTimeout)

var _ = AfterSuite(func(done Done) {
	Expect(env.Stop()).NotTo(HaveOccurred())
	close(done)
}, envtest.StopTimeout)

var _ = Describe("Test", func() {

	var c client.Client
	var stop chan struct{}

	// every test case gets it's own manager and client instances. not sure if that's the best
	// practice but it  seems to be fast enough.
	BeforeEach(func() {
		// 1. initializing manager
		logf.Log.Info("test.BeforeEach: initializing manager")
		mgr, err := manager.New(env.Config, manager.Options{
			Port:    env.WebhookInstallOptions.LocalServingPort,
			Host:    env.WebhookInstallOptions.LocalServingHost,
			CertDir: env.WebhookInstallOptions.LocalServingCertDir,
		})
		Expect(err).NotTo(HaveOccurred())

		// 2. initializing scheme with v1beta1 types
		logf.Log.Info("test.BeforeEach: initializing scheme")
		err = apis.AddToScheme(mgr.GetScheme())
		Expect(err).NotTo(HaveOccurred())

		// 3. registering instance admission controller
		logf.Log.Info("test.BeforeEach: initializing webhook server")
		server := mgr.GetWebhookServer()
		server.Register(instanceAdmissionWebhookPath, &ctrhook.Admission{Handler: &InstanceAdmission{}})

		// 4. starting the manager
		stop = make(chan struct{})
		go func() {
			err = mgr.Start(stop)
			Expect(err).NotTo(HaveOccurred())
		}()

		// 5. creating the client. **Note:** client.New method will create an uncached client, a cached one
		// (e.g. mgr.GetClient) leads to caching issues in this test.
		logf.Log.Info("test.BeforeEach: initializing client")
		c, err = client.New(env.Config, client.Options{Scheme: mgr.GetScheme()})
		Expect(err).NotTo(HaveOccurred())

		// --- DEBUG ---
		// write kubconfig to be able to debug the cluster with kubectl
		f, err := os.Create("testkubeconfig")
		defer f.Close()
		Expect(err).NotTo(HaveOccurred())

		err = testutils.Kubeconfig(env.Config, f)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		close(stop)
	})

	deploy := v1beta1.DeployPlanName
	update := v1beta1.UpdatePlanName
	cleanup := v1beta1.CleanupPlanName
	//empty := ""

	ov := &v1beta1.OperatorVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-operator", Namespace: "default"},
		TypeMeta:   metav1.TypeMeta{Kind: "OperatorVersion", APIVersion: "kudo.dev/v1beta1"},
		Spec: v1beta1.OperatorVersionSpec{
			Plans: map[string]v1beta1.Plan{deploy: {}, update: {}, cleanup: {}},
			Parameters: []v1beta1.Parameter{
				{
					Name:    "foo",
					Trigger: deploy,
				},
				{
					Name:    "other-foo",
					Trigger: deploy,
				},
				{
					Name:    "bar",
					Trigger: update,
				},
			},
		},
	}

	idle := &v1beta1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo-instance",
			Namespace: "default",
		},
		Spec: v1beta1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{Name: "foo-operator"},
		},
		Status: v1beta1.InstanceStatus{},
	}

	Describe("Instance admission controller", func() {
		It("should schedule deploy plan when an instance is created", func(done Done) {

			//Eventually(func() error {
			//	return c.Create(context.TODO(), ov)
			//}, 1*time.Second).Should(BeNil())

			// 1. create the OV first
			err := c.Create(context.TODO(), ov)
			Expect(err).NotTo(HaveOccurred())

			// 2. create the initial Instance
			err = c.Create(context.TODO(), idle)
			Expect(err).NotTo(HaveOccurred())

			// 3. admission controller should schedule 'deploy' plan and add cleanup finalizer
			key := keyFrom(idle)
			i := instanceWith(c, key)

			Expect(i.HasPlanScheduled(deploy))
			Expect(i.HasCleanupFinalizer()).Should(BeTrue())

			uid := i.Spec.PlanExecution.UID // save current plan UID for later checks

			// 4. finish the 'deploy' plan by updating the corresponding status field
			i.Status.PlanStatus = map[string]v1beta1.PlanStatus{
				deploy: {Name: deploy, UID: uid, Status: v1beta1.ExecutionComplete},
			}
			err = c.Status().Update(context.TODO(), i)
			Expect(err).NotTo(HaveOccurred())

			// 5. admission controller should reset the Spec.PlanExecution

			Eventually(func() bool {
				i = instanceWith(c, key)
				s, _ := json.MarshalIndent(i, "", "  ")
				log.Printf(">>> DEBUG TEST:\n%s", s)

				return i.HasPlanScheduled(deploy)
			}, 3*time.Second).Should(BeFalse())

			//s, _ := json.MarshalIndent(i, "", "  ")
			//log.Printf("+++ DEBUG: foo-instance +++\n%s", s)

			close(done)
		}, 1000000)
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
func instanceWith(c client.Client, key client.ObjectKey) *v1beta1.Instance {
	i := &v1beta1.Instance{}
	err := c.Get(context.TODO(), key, i)
	Expect(err).NotTo(HaveOccurred())
	return i
}
