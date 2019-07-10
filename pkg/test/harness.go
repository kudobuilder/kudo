package test

import (
	"context"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/controller"
	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
	"github.com/kudobuilder/kudo/pkg/webhook"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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
		timeout = 30
	}
	return timeout
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
	if h.config != nil {
		return h.config, nil
	}

	var err error

	if h.TestSuite.StartControlPlane {
		h.config, err = h.RunTestEnv()
	} else {
		h.config, err = config.GetConfig()
	}

	return h.config, err
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
	if h.client != nil && !forceNew {
		return h.client, nil
	}

	config, err := h.Config()
	if err != nil {
		return nil, err
	}

	h.client, err = client.New(config, client.Options{
		Scheme: testutils.Scheme(),
	})
	return h.client, err
}

// DiscoveryClient returns the current Kubernetes discovery client for the test harness.
func (h *Harness) DiscoveryClient() (discovery.DiscoveryInterface, error) {
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

// RunTests should be called from within a Go test (t) and launches all of the KUDO integration
// tests at dir.
func (h *Harness) RunTests() {
	cl, err := h.Client(false)
	if err != nil {
		h.T.Fatal(err)
	}

	dclient, err := h.DiscoveryClient()
	if err != nil {
		h.T.Fatal(err)
	}

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
			test.Client = cl
			test.DiscoveryClient = dclient

			if err := test.LoadTestSteps(); err != nil {
				t.Fatal(err)
			}

			t.Run(test.Name, test.TestCaseFactory())
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
	crds, err := testutils.InstallManifests(context.TODO(), cl, dClient, h.TestSuite.CRDDir)
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
	if _, err := testutils.InstallManifests(context.TODO(), cl, dClient, h.TestSuite.ManifestsDir); err != nil {
		h.T.Fatal(err)
	}

	if h.TestSuite.StartKUDO || h.TestSuite.StartControlPlane {
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

	if h.env != nil {
		h.env.Stop()
		h.env = nil
	}
}
