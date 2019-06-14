package test

import (
	"context"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/kudobuilder/kudo/pkg/apis"
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
	T                 *testing.T
	CRDDir            string
	ManifestsDir      string
	TestDirs          []string
	StartControlPlane bool
	StartKUDO         bool

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
			Steps: []*Step{},
			Name:  file.Name(),
			Dir:   filepath.Join(dir, file.Name()),
		})
	}

	return tests, nil
}

// RunTestEnv starts a Kubernetes API server and etcd server for use in the
// tests and returns the Kubernetes configuration.
func (h *Harness) RunTestEnv() (*rest.Config, error) {
	h.env = &envtest.Environment{}

	config, err := h.env.Start()
	if err != nil {
		return nil, err
	}

	return config, nil
}

// Config returns the current Kubernetes configuration - either from the environment
// or from the created temporary control plane.
func (h *Harness) Config() (*rest.Config, error) {
	if h.config != nil {
		return h.config, nil
	}

	var err error

	if h.StartControlPlane {
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

	mgr, err := manager.New(config, manager.Options{})
	if err != nil {
		return err
	}

	if err = apis.AddToScheme(mgr.GetScheme()); err != nil {
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
func (h *Harness) Client() (client.Client, error) {
	if h.client != nil {
		return h.client, nil
	}

	config, err := h.Config()
	if err != nil {
		return nil, err
	}

	h.client, err = client.New(config, client.Options{})
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
	cl, err := h.Client()
	if err != nil {
		h.T.Fatal(err)
	}

	dClient, err := h.DiscoveryClient()
	if err != nil {
		h.T.Fatal(err)
	}

	tests := []*Case{}

	for _, testDir := range h.TestDirs {
		tempTests, err := h.LoadTests(testDir)
		if err != nil {
			h.T.Fatal(err)
		}
		tests = append(tests, tempTests...)
	}

	h.T.Run("harness", func(t *testing.T) {
		for _, test := range tests {
			test.Client = cl
			test.DiscoveryClient = dClient

			if err := test.LoadTestSteps(); err != nil {
				t.Fatal(err)
			}

			t.Run(test.Name, test.TestCaseFactory())
		}
	})
}

// Run the test harness - start KUDO and the control plane and install the frameworks, if necessary
// and then run the tests.
func (h *Harness) Run() {
	rand.Seed(time.Now().UTC().UnixNano())

	defer h.Stop()

	if h.CRDDir != "" {
		if err := h.installManifests(h.CRDDir); err != nil {
			h.T.Fatal(err)
		}

		h.client = nil
	}

	if h.StartKUDO || h.StartControlPlane {
		if err := h.RunKUDO(); err != nil {
			h.T.Fatal(err)
		}
	}

	if err := h.installManifests(h.ManifestsDir); err != nil {
		h.T.Fatal(err)
	}

	h.RunTests()
}

// Stop the test environment and KUDO, clean up the harness.
func (h *Harness) Stop() {
	if h.env != nil {
		h.env.Stop()
		h.env = nil
	}

	if h.managerStopCh != nil {
		close(h.managerStopCh)
		h.managerStopCh = nil
	}
}

// installManifests recurses the path provided to install all configured resources into Kubernetes.
func (h *Harness) installManifests(path string) error {
	cl, err := h.Client()
	if err != nil {
		return err
	}

	dClient, err := h.DiscoveryClient()
	if err != nil {
		return err
	}

	return testutils.InstallManifests(context.TODO(), cl, dClient, path)
}
