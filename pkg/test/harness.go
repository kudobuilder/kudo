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

	volumetypes "github.com/docker/docker/api/types/volume"
	docker "github.com/docker/docker/client"
	yaml "gopkg.in/yaml.v2"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	kindConfig "sigs.k8s.io/kind/pkg/apis/config/v1alpha3"
	kind "sigs.k8s.io/kind/pkg/cluster"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/controller/instance"
	"github.com/kudobuilder/kudo/pkg/controller/operator"
	"github.com/kudobuilder/kudo/pkg/controller/operatorversion"
	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
)

// Harness loads and runs tests based on the configuration provided.
type Harness struct {
	TestSuite kudo.TestSuite
	T         *testing.T

	logger         testutils.Logger
	managerStopCh  chan struct{}
	config         *rest.Config
	docker         testutils.DockerClient
	client         client.Client
	dclient        discovery.DiscoveryInterface
	env            *envtest.Environment
	kind           *kind.Provider
	kubeConfigPath string
	clientLock     sync.Mutex
	configLock     sync.Mutex
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

	timeout := h.GetTimeout()
	h.T.Logf("Going to run test suite with timeout of %d seconds for each step", timeout)

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		tests = append(tests, &Case{
			Timeout:    timeout,
			Steps:      []*Step{},
			Name:       file.Name(),
			Dir:        filepath.Join(dir, file.Name()),
			SkipDelete: h.TestSuite.SkipDelete,
		})
	}

	return tests, nil
}

// GetLogger returns an initialized test logger.
func (h *Harness) GetLogger() testutils.Logger {
	if h.logger == nil {
		h.logger = testutils.NewTestLogger(h.T, "")
	}

	return h.logger
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
	if h.kind == nil {
		h.kind = kind.NewProvider()

		contexts, err := h.kind.List()
		if err != nil {
			return nil, err
		}

		for _, context := range contexts {
			// There is already a cluster with this context, let's re-use it.
			if context == h.TestSuite.KINDContext {
				return clientcmd.BuildConfigFromFlags("", h.explicitPath())
			}
		}

		kindCfg := &kindConfig.Cluster{}

		if h.TestSuite.KINDConfig != "" {
			var err error
			kindCfg, err = loadKindConfig(h.TestSuite.KINDConfig)
			if err != nil {
				return nil, err
			}
		}

		if err := h.addNodeCaches(kindCfg); err != nil {
			return nil, err
		}

		h.kubeConfigPath, err = ioutil.TempDir("", "kudo")
		if err != nil {
			return nil, err
		}

		if err := h.kind.Create(
			h.TestSuite.KINDContext,
			kind.CreateWithV1Alpha3Config(kindCfg),
			kind.CreateWithKubeconfigPath(h.explicitPath()),
		); err != nil {
			return nil, err
		}
	}

	return clientcmd.BuildConfigFromFlags("", h.explicitPath())
}

func (h *Harness) addNodeCaches(kindCfg *kindConfig.Cluster) error {
	if !h.TestSuite.KINDNodeCache {
		return nil
	}

	dockerClient, err := h.DockerClient()
	if err != nil {
		return err
	}

	// Determine the correct API version to use with the user's Docker client.
	dockerClient.NegotiateAPIVersion(context.TODO())

	// add a default node if there are none specified.
	if len(kindCfg.Nodes) == 0 {
		kindCfg.Nodes = append(kindCfg.Nodes, kindConfig.Node{})
	}

	if h.TestSuite.KINDContext == "" {
		h.TestSuite.KINDContext = kudo.DefaultKINDContext
	}

	for index := range kindCfg.Nodes {
		volume, err := dockerClient.VolumeCreate(context.TODO(), volumetypes.VolumeCreateBody{
			Driver: "local",
			Name:   fmt.Sprintf("%s-%d", h.TestSuite.KINDContext, index),
		})
		if err != nil {
			h.T.Log("error creating volume for node", err)
			continue
		}

		h.T.Log("node mount point", volume.Mountpoint)
		kindCfg.Nodes[index].ExtraMounts = append(kindCfg.Nodes[index].ExtraMounts, kindConfig.Mount{
			ContainerPath: "/var/lib/containerd",
			HostPath:      volume.Mountpoint,
		})
	}

	return nil
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

	// Setup all Controllers

	h.logger.Log("Setting up operator controller")
	err = (&operator.Reconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr)
	if err != nil {
		h.logger.Log(err, "unable to register operator controller to the manager")
		os.Exit(1)
	}

	h.logger.Log("Setting up operator version controller")
	err = (&operatorversion.Reconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr)
	if err != nil {
		h.logger.Log(err, "unable to register operator controller to the manager")
		os.Exit(1)
	}

	h.logger.Log("Setting up instance controller")
	err = (&instance.Reconciler{
		Client:   mgr.GetClient(),
		Recorder: mgr.GetEventRecorderFor("instance-controller"),
		Scheme:   mgr.GetScheme(),
	}).SetupWithManager(mgr)
	if err != nil {
		h.logger.Log(err, "unable to register instance controller to the manager")
		os.Exit(1)
	}

	h.managerStopCh = make(chan struct{})
	go func(stopCh chan struct{}) {
		if err := mgr.Start(stopCh); err != nil {
			fmt.Printf("failed to start the manager")
			os.Exit(-1)
		}
	}(h.managerStopCh)

	return nil
}

// Client returns the current Kubernetes client for the test harness.
func (h *Harness) Client(forceNew bool) (client.Client, error) {
	h.clientLock.Lock()
	defer h.clientLock.Unlock()

	if h.client != nil && !forceNew {
		return h.client, nil
	}

	cfg, err := h.Config()
	if err != nil {
		return nil, err
	}

	h.client, err = testutils.NewRetryClient(cfg, client.Options{
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

	cfg, err := h.Config()
	if err != nil {
		return nil, err
	}

	h.dclient, err = discovery.NewDiscoveryClientForConfig(cfg)
	return h.dclient, err
}

// DockerClient returns the Docker client to use for the test harness.
func (h *Harness) DockerClient() (testutils.DockerClient, error) {
	if h.docker != nil {
		return h.docker, nil
	}

	var err error
	h.docker, err = docker.NewClientWithOpts(docker.FromEnv)
	return h.docker, err
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

			t.Run(test.Name, func(t *testing.T) {
				test.Logger = testutils.NewTestLogger(t, test.Name)

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

	if err := testutils.RunCommands(h.GetLogger(), "default", "", h.TestSuite.Commands, ""); err != nil {
		h.T.Fatal(err)
	}

	if err := testutils.RunKubectlCommands(h.GetLogger(), "default", h.TestSuite.Kubectl, ""); err != nil {
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

		if err := h.kind.CollectLogs(h.TestSuite.KINDContext, logDir); err != nil {
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
		if err := h.kind.Delete(h.TestSuite.KINDContext, h.explicitPath()); err != nil {
			h.T.Log("error tearing down kind cluster", err)
		}

		if err := os.RemoveAll(h.kubeConfigPath); err != nil {
			h.T.Log("error removing temporary directory", err)
		}

		h.kind = nil
	}
}

func (h *Harness) explicitPath() string {
	return filepath.Join(h.kubeConfigPath, "kubeconfig")
}

func loadKindConfig(path string) (*kindConfig.Cluster, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cluster := &kindConfig.Cluster{}

	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.SetStrict(true)

	if err := decoder.Decode(cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}
