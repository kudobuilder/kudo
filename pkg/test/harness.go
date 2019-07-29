package test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/controller"
	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
	"github.com/kudobuilder/kudo/pkg/webhook"
	"github.com/pkg/errors"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	kind "sigs.k8s.io/kind/pkg/cluster"
	kindConfig "sigs.k8s.io/kind/pkg/cluster/config"
)

// Harness loads and runs tests based on the configuration provided.
type Harness struct {
	TestSuite kudo.TestSuite
	T         *testing.T

	managerStopCh chan struct{}
	config        *rest.Config
	client        client.Client
	dclient       discovery.DiscoveryInterface
	env           *envtest.Environment
	kind          *kind.Context
	clientLock    sync.Mutex
	configLock    sync.Mutex
}

// LoadTests loads all of the tests in a given directory.
func (h *Harness) LoadTests(dir string) ([]*Case, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	tests := []*Case{}

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		tests = append(tests, &Case{
			Timeout:    h.GetTimeout(),
			Steps:      []*Step{},
			Name:       file.Name(),
			Dir:        filepath.Join(dir, file.Name()),
			SkipDelete: h.TestSuite.SkipDelete,
		})
	}

	return tests, nil
}

// GetTimeout returns the configured timeout for the test suite.
func (h *Harness) GetTimeout() int {
	timeout := 30
	if h.TestSuite.Timeout != 0 {
		timeout = h.TestSuite.Timeout
	}
	return timeout
}

// RunKIND starts a KIND cluster.
func (h *Harness) RunKIND() (*rest.Config, error) {
	contexts, err := kind.List()
	if err != nil {
		return nil, err
	}

	for index := range contexts {
		if contexts[index].Name() != h.TestSuite.KINDContext {
			continue
		}

		h.kind = &contexts[index]
		break
	}

	if h.kind == nil {
		h.kind = kind.NewContext(h.TestSuite.KINDContext)

		kindCfg := &kindConfig.Cluster{}

		if h.TestSuite.KINDConfig != "" {
			objs, err := testutils.LoadYAML(h.TestSuite.KINDConfig)
			if err != nil {
				return nil, err
			}

			var ok bool
			kindCfg, ok = objs[0].(*kindConfig.Cluster)
			if !ok {
				return nil, fmt.Errorf("kind configuration contains invalid kind config file")
			}
		}

		err := h.kind.Create(kindCfg)
		if err != nil {
			return nil, err
		}
	}

	return clientcmd.BuildConfigFromFlags("", h.kind.KubeConfigPath())
}

// RunTestEnv starts a Kubernetes API server and etcd server for use in the
// tests and returns the Kubernetes configuration.
func (h *Harness) RunTestEnv() (*rest.Config, error) {
	started := time.Now()

	testenv, err := testutils.StartTestEnvironment()
	if err != nil {
		return nil, err
	}

	h.T.Log("started test environment (kube-apiserver and etcd) in", time.Since(started))
	h.env = testenv.Environment

	return testenv.Config, nil
}

// Config returns the current Kubernetes configuration - either from the environment
// or from the created temporary control plane.
func (h *Harness) Config() (*rest.Config, error) {
	h.configLock.Lock()
	defer h.configLock.Unlock()

	if h.config != nil {
		return h.config, nil
	}

	var err error

	if h.TestSuite.StartControlPlane {
		h.T.Log("Running tests with a mocked control plane (kube-apiserver and etcd).")
		h.config, err = h.RunTestEnv()
	} else if h.TestSuite.StartKIND {
		h.T.Log("Running tests with KIND.")
		h.config, err = h.RunKIND()
	} else {
		h.T.Log("Running tests using configured kubeconfig.")
		h.config, err = config.GetConfig()
	}

	if err != nil {
		return h.config, err
	}

	f, err := os.Create("kubeconfig")
	if err != nil {
		return h.config, err
	}

	defer f.Close()

	return h.config, testutils.Kubeconfig(h.config, f)
}

// RunKUDO starts the KUDO controllers and installs the required CRDs.
func (h *Harness) RunKUDO() error {
	config, err := h.Config()
	if err != nil {
		return err
	}

	mgr, err := manager.New(config, manager.Options{
		Scheme: testutils.Scheme(),
	})
	if err != nil {
		return err
	}

	if err = controller.AddToManager(mgr); err != nil {
		return err
	}

	if err = webhook.AddToManager(mgr); err != nil {
		return err
	}

	h.managerStopCh = make(chan struct{})
	go mgr.Start(h.managerStopCh)

	return nil
}

// Client returns the current Kubernetes client for the test harness.
func (h *Harness) Client(forceNew bool) (client.Client, error) {
	h.clientLock.Lock()
	defer h.clientLock.Unlock()

	if h.client != nil && !forceNew {
		return h.client, nil
	}

	config, err := h.Config()
	if err != nil {
		return nil, err
	}

	h.client, err = testutils.NewRetryClient(config, client.Options{
		Scheme: testutils.Scheme(),
	})
	return h.client, err
}

// DiscoveryClient returns the current Kubernetes discovery client for the test harness.
func (h *Harness) DiscoveryClient() (discovery.DiscoveryInterface, error) {
	h.clientLock.Lock()
	defer h.clientLock.Unlock()

	if h.dclient != nil {
		return h.dclient, nil
	}

	config, err := h.Config()
	if err != nil {
		return nil, err
	}

	h.dclient, err = discovery.NewDiscoveryClientForConfig(config)
	return h.dclient, err
}

// DoKubectl runs all kubectl commands defined for the test suite.
func (h *Harness) DoKubectl() error {
	if h.TestSuite.Kubectl == nil {
		return nil
	}

	for _, cmd := range h.TestSuite.Kubectl {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}

		h.T.Log("Running kubectl:", cmd)

		if err := testutils.Kubectl(context.TODO(), "default", cmd, "", stdout, stderr); err != nil {
			return errors.Wrap(err, stderr.String())
		}

		h.T.Log(stdout.String())
	}

	return nil
}

// RunTests should be called from within a Go test (t) and launches all of the KUDO integration
// tests at dir.
func (h *Harness) RunTests() {
	tests := []*Case{}

	for _, testDir := range h.TestSuite.TestDirs {
		tempTests, err := h.LoadTests(testDir)
		if err != nil {
			h.T.Fatal(err)
		}
		tests = append(tests, tempTests...)
	}

	h.T.Run("harness", func(t *testing.T) {
		for _, test := range tests {
			test.Client = h.Client
			test.DiscoveryClient = h.DiscoveryClient
			test.Logger = testutils.NewTestLogger(t, test.Name)

			t.Run(test.Name, func(t *testing.T) {
				if err := test.LoadTestSteps(); err != nil {
					t.Fatal(err)
				}

				test.Run(t)
			})
		}
	})
}

// Run the test harness - start KUDO and the control plane and install the operators, if necessary
// and then run the tests.
func (h *Harness) Run() {
	rand.Seed(time.Now().UTC().UnixNano())

	defer h.Stop()

	cl, err := h.Client(false)
	if err != nil {
		h.T.Fatal(err)
	}

	dClient, err := h.DiscoveryClient()
	if err != nil {
		h.T.Fatal(err)
	}

	// Install CRDs
	crdKind := testutils.NewResource("apiextensions.k8s.io/v1beta1", "CustomResourceDefinition", "", "")
	crds, err := testutils.InstallManifests(context.TODO(), cl, dClient, h.TestSuite.CRDDir, crdKind)
	if err != nil {
		h.T.Fatal(err)
	}

	if err := testutils.WaitForCRDs(dClient, crds); err != nil {
		h.T.Fatal(err)
	}

	// Create a new client to bust the client's CRD cache.
	cl, err = h.Client(true)
	if err != nil {
		h.T.Fatal(err)
	}

	// Install required manifests.
	for _, manifestDir := range h.TestSuite.ManifestDirs {
		if _, err := testutils.InstallManifests(context.TODO(), cl, dClient, manifestDir); err != nil {
			h.T.Fatal(err)
		}
	}

	if err := h.DoKubectl(); err != nil {
		h.T.Fatal(err)
	}

	if h.TestSuite.StartKUDO {
		if err := h.RunKUDO(); err != nil {
			h.T.Fatal(err)
		}
	}

	h.RunTests()
}

// Stop the test environment and KUDO, clean up the harness.
func (h *Harness) Stop() {
	if h.managerStopCh != nil {
		close(h.managerStopCh)
		h.managerStopCh = nil
	}

	if h.kind != nil {
		logDir := filepath.Join(h.TestSuite.ArtifactsDir, fmt.Sprintf("kind-logs-%d", time.Now().Unix()))

		h.T.Log("collecting cluster logs to", logDir)

		if err := h.kind.CollectLogs(logDir); err != nil {
			h.T.Log("error collecting kind cluster logs", err)
		}
	}

	if h.TestSuite.SkipClusterDelete || h.TestSuite.SkipDelete {
		cwd, _ := os.Getwd()
		kubeconfig := filepath.Join(cwd, "kubeconfig")

		h.T.Log("skipping cluster tear down")
		h.T.Log(fmt.Sprintf("to connect to the cluster, run: export KUBECONFIG=\"%s\"", kubeconfig))

		return
	}

	if h.env != nil {
		h.T.Log("tearing down mock control plane")
		if err := h.env.Stop(); err != nil {
			h.T.Log("error tearing down mock control plane", err)
		}

		h.env = nil
	}

	if h.kind != nil {
		h.T.Log("tearing down kind cluster")
		if err := h.kind.Delete(); err != nil {
			h.T.Log("error tearing down kind cluster", err)
		}

		h.kind = nil
	}
}
