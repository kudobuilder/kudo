// +build integration

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	testutils "github.com/kudobuilder/kuttl/pkg/test/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/crd"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/prereq"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

var testenv testutils.TestEnvironment

func TestMain(m *testing.M) {
	var err error

	testenv, err = testutils.StartTestEnvironment(testutils.APIServerDefaultArgs, false)
	if err != nil {
		log.Fatal(err)
	}

	exitCode := m.Run()
	err = testenv.Environment.Stop()
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(exitCode)
}

const (
	operatorFileName        = "kudo.dev_operators.yaml"
	operatorVersionFileName = "kudo.dev_operatorversions.yaml"
	instanceFileName        = "kudo.dev_instances.yaml"
	manifestsDir            = "../../../config/crds/"
)

func TestKudoClientValidate(t *testing.T) {
	tests := []struct {
		err string
	}{
		{"CRDs invalid: CRD operators.kudo.dev is not installed"}, // verify that NewClient tries to validate CRDs
	}

	for _, tt := range tests {
		_, err := kudo.NewClientForConfig(testenv.Config, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), tt.err)
	}
}

func TestCrds_Config(t *testing.T) {
	crds := crd.NewInitializer()
	assertManifestFileMatch(t, operatorFileName, crds.Operator)
	assertManifestFileMatch(t, operatorVersionFileName, crds.OperatorVersion)
	assertManifestFileMatch(t, instanceFileName, crds.Instance)
}

func assertManifestFileMatch(t *testing.T, fileName string, expectedObject runtime.Object) {
	expectedContent, err := runtimeObjectAsBytes(expectedObject)
	assert.NoError(t, err)
	path := filepath.Join(manifestsDir, fileName)
	of, err := ioutil.ReadFile(path)
	assert.NoError(t, err)

	assert.Equal(t, string(expectedContent), string(of), fmt.Sprintf("embedded file %s does not match the source, run 'make generate'", fileName))
}

func assertStringContains(t *testing.T, expected string, actual string) {
	assert.True(t, strings.Contains(actual, expected), "Expected to find '%s' in '%s'", expected, actual)
}

func runtimeObjectAsBytes(o runtime.Object) ([]byte, error) {
	bytes, err := yaml.Marshal(o)
	if err != nil {
		return nil, err
	}
	return append([]byte("\n---\n"), bytes...), nil
}

func TestIntegInitForCRDs(t *testing.T) {
	// Kubernetes client caches the types, so we need to re-initialize it.
	testClient, err := testutils.NewRetryClient(testenv.Config, client.Options{
		Scheme: testutils.Scheme(),
	})
	assert.NoError(t, err)
	kclient := getKubeClient(t)

	instance := testutils.NewResource("kudo.dev/v1beta1", "Instance", "zk", "default")
	// Verify that we cannot create the instance, because the test environment is empty.
	assert.IsType(t, &meta.NoKindMatchError{}, testClient.Create(context.TODO(), instance))

	// Install all of the CRDs.
	crds := crd.NewInitializer().Resources()

	var buf bytes.Buffer
	var errBuf bytes.Buffer
	cmd := &initCmd{
		out:     &buf,
		errOut:  &errBuf,
		fs:      afero.NewMemMapFs(),
		client:  kclient,
		crdOnly: true,
		version: "dev",
	}
	err = cmd.run()
	assert.NoError(t, err)
	// WaitForCRDs to be created... the init cmd did NOT wait
	assert.NoError(t, envtest.WaitForCRDs(testenv.Config, crds, envtest.CRDInstallOptions{
		PollInterval: 100 * time.Millisecond,
		MaxTime:      10 * time.Second,
	}))
	defer func() {
		assert.NoError(t, deleteObjects(crds, testClient))
	}()

	// Kubernetes client caches the types, so we need to re-initialize it.
	testClient, err = testutils.NewRetryClient(testenv.Config, client.Options{
		Scheme: testutils.Scheme(),
	})
	assert.NoError(t, err)

	// make sure that we can create an object of this type now
	assert.NoError(t, testClient.Create(context.TODO(), instance))
	assert.NoError(t, testClient.Delete(context.TODO(), instance))
}

func TestIntegInitWithNameSpace(t *testing.T) {
	namespace := "integration-test"
	// Kubernetes client caches the types, so we need to re-initialize it.
	testClient, err := testutils.NewRetryClient(testenv.Config, client.Options{
		Scheme: testutils.Scheme(),
	})
	assert.NoError(t, err)
	kclient := getKubeClient(t)

	instance := testutils.NewResource("kudo.dev/v1beta1", "Instance", "zk", "ns")
	// Verify that we cannot create the instance, because the test environment is empty.
	err = testClient.Create(context.TODO(), instance)
	assert.Error(t, err, "Expected an Error but got none")

	// Install all of the CRDs.
	crds := crd.NewInitializer().Resources()

	var buf bytes.Buffer
	var errBuf bytes.Buffer
	cmd := &initCmd{
		out:                 &buf,
		errOut:              &errBuf,
		fs:                  afero.NewMemMapFs(),
		client:              kclient,
		ns:                  namespace,
		selfSignedWebhookCA: true,
		version:             "dev",
	}

	// On first attempt, the namespace does not exist, so the error is expected.
	err = cmd.run()
	require.Error(t, err)
	assert.Equal(t, "failed to verify installation requirements", err.Error())
	assertStringContains(t, "Namespace integration-test does not exist - KUDO expects that any namespace except the default kudo-system is created beforehand", errBuf.String())

	// Then we manually create the namespace.
	ns := testutils.NewResource("v1", "Namespace", namespace, "")
	assert.NoError(t, testClient.Create(context.TODO(), ns))
	defer func() {
		assert.NoError(t, testClient.Delete(context.TODO(), ns))
	}()

	// On second attempt run should succeed.
	err = cmd.run()
	assert.NoError(t, err, buf.String())
	defer func() {
		assert.NoError(t, deleteInitPrereqs(cmd, testClient))
	}()

	// WaitForCRDs to be created... the init cmd did NOT wait
	assert.NoError(t, envtest.WaitForCRDs(testenv.Config, crds, envtest.CRDInstallOptions{
		PollInterval: 100 * time.Millisecond,
		MaxTime:      10 * time.Second,
	}))

	// make sure that the controller lives in the correct namespace
	kclient = getKubeClient(t)
	statefulsets, err := kclient.KubeClient.
		AppsV1().
		StatefulSets(namespace).
		List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)

	kudoControllerFound := false
	for _, ss := range statefulsets.Items {
		if ss.Name == kudoinit.DefaultManagerName {
			kudoControllerFound = true
		}
	}
	assert.True(t, kudoControllerFound, fmt.Sprintf("No kudo-controller-manager statefulset found in namespace %s", namespace))
}

/*
	Test the below 5 scenarios
		1. Run Init command with a serviceAccount that is not present in the cluster.
		2. Run init command with a serviceAccount that is present in the cluster, but not in the clusterrole-binding.
		3. Run Init command with a serviceAccount that does not have cluster-admin role.
		4. Run Init command with a serviceAccount that does have cluster-admin role, but not in the expected namespace.
		5. Run Init command with a serviceAccount that is present in the cluster and also has cluster-admin role.
*/
func TestInitWithServiceAccount(t *testing.T) {
	tests := []struct {
		name               string
		serviceAccount     string
		roleBindingRole    string
		roleBindingNs      string
		errMessageContains string
	}{
		{
			name:               "service account not present",
			serviceAccount:     "",
			errMessageContains: "Service Account test-account does not exists - KUDO expects the serviceAccount to be present in the namespace sa-integration-test-0",
		},
		{
			name:               "service account has no rb",
			serviceAccount:     "test-account",
			errMessageContains: "Service Account test-account does not have cluster-admin role - KUDO expects the serviceAccount passed to be in the namespace sa-integration-test-1 and to have cluster-admin role",
		},
		{
			name:               "rb has no cluster-admin role",
			serviceAccount:     "test-account",
			roleBindingRole:    "not-admin",
			errMessageContains: "Service Account test-account does not have cluster-admin role - KUDO expects the serviceAccount passed to be in the namespace sa-integration-test-2 and to have cluster-admin role",
		},
		{
			name:               "rb has different ns",
			serviceAccount:     "test-account",
			roleBindingRole:    "not-admin",
			roleBindingNs:      "otherns",
			errMessageContains: "Service Account test-account does not have cluster-admin role - KUDO expects the serviceAccount passed to be in the namespace sa-integration-test-3 and to have cluster-admin role",
		},
		{
			name:               "rb has admin in different ns",
			serviceAccount:     "test-account",
			roleBindingRole:    "cluster-admin",
			roleBindingNs:      "otherns",
			errMessageContains: "Service Account test-account does not have cluster-admin role - KUDO expects the serviceAccount passed to be in the namespace sa-integration-test-4 and to have cluster-admin role",
		},
		{
			name:               "rb has cluster-admin role",
			serviceAccount:     "test-account",
			roleBindingRole:    "cluster-admin",
			errMessageContains: "",
		},
	}

	namespaceBase := "sa-integration-test"

	for idx, tt := range tests {
		idx := idx
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			namespace := fmt.Sprintf("%s-%d", namespaceBase, idx)

			// Kubernetes client caches the types, so we need to re-initialize it.
			testClient, err := testutils.NewRetryClient(testenv.Config, client.Options{
				Scheme: testutils.Scheme(),
			})
			assert.NoError(t, err)
			kclient := getKubeClient(t)

			instance := testutils.NewResource("kudo.dev/v1beta1", "Instance", "zk", "ns")
			// Verify that we cannot create the instance, because the test environment is empty.
			assert.IsType(t, &meta.NoKindMatchError{}, testClient.Create(context.TODO(), instance))

			// Install all of the CRDs.
			crds := crd.NewInitializer().Resources()

			var buf bytes.Buffer
			var errBuf bytes.Buffer
			cmd := &initCmd{
				out:                 &buf,
				errOut:              &errBuf,
				fs:                  afero.NewMemMapFs(),
				client:              kclient,
				ns:                  namespace,
				serviceAccount:      "test-account",
				selfSignedWebhookCA: true,
				version:             "dev",
			}

			ns := testutils.NewResource("v1", "Namespace", namespace, "")
			assert.NoError(t, testClient.Create(context.TODO(), ns))
			defer func() {
				assert.NoError(t, testClient.Delete(context.TODO(), ns))
			}()

			if tt.serviceAccount != "" {
				sa2 := testutils.NewResource("v1", "ServiceAccount", tt.serviceAccount, namespace)
				assert.NoError(t, testClient.Create(context.TODO(), sa2))
				defer func() {
					assert.NoError(t, testClient.Delete(context.TODO(), sa2))
				}()
			}

			if tt.roleBindingRole != "" {
				rbNamespace := tt.roleBindingNs
				if rbNamespace == "" {
					rbNamespace = namespace
				}
				crb := testutils.NewClusterRoleBinding("rbac.authorization.k8s.io/v1", "ClusterRoleBinding", "kudo-clusterrole-binding", rbNamespace, tt.serviceAccount, tt.roleBindingRole)
				assert.NoError(t, testClient.Create(context.TODO(), crb))
				defer func() {
					assert.NoError(t, testClient.Delete(context.TODO(), crb))
				}()
			}

			err = cmd.run()

			if tt.errMessageContains != "" {
				require.Error(t, err)
				assert.Equal(t, "failed to verify installation requirements", err.Error())
				assertStringContains(t, tt.errMessageContains, errBuf.String())
			} else {
				assert.NoError(t, err)
				defer func() {
					assert.NoError(t, deleteInitPrereqs(cmd, testClient))
				}()

				// WaitForCRDs to be created... the init cmd did NOT wait
				assert.NoError(t, envtest.WaitForCRDs(testenv.Config, crds, envtest.CRDInstallOptions{
					PollInterval: 100 * time.Millisecond,
					MaxTime:      10 * time.Second,
				}))

				// make sure that the controller lives in the correct namespace
				kclient = getKubeClient(t)
				statefulsets, err := kclient.KubeClient.
					AppsV1().
					StatefulSets(namespace).
					List(context.TODO(), metav1.ListOptions{})
				assert.NoError(t, err)

				kudoControllerFound := false
				for _, ss := range statefulsets.Items {
					if ss.Name == kudoinit.DefaultManagerName {
						kudoControllerFound = true
					}
				}
				assert.True(t, kudoControllerFound, fmt.Sprintf("No kudo-controller-manager statefulset found in namespace %s", namespace))
			}
		})
	}
}

func TestReInitFails(t *testing.T) {
	//	 if the CRD exists and we init again there should be no error
	testClient, err := testutils.NewRetryClient(testenv.Config, client.Options{
		Scheme: testutils.Scheme(),
	})
	assert.Nil(t, err)
	kclient := getKubeClient(t)

	instance := testutils.NewResource("kudo.dev/v1beta1", "Instance", "zk", "ns")
	// Verify that we cannot create the instance, because the test environment is empty.
	assert.IsType(t, &meta.NoKindMatchError{}, testClient.Create(context.TODO(), instance))

	// Install all of the CRDs.
	crds := crd.NewInitializer().Resources()

	var buf bytes.Buffer
	clog.InitNoFlag(&buf, clog.Level(4))
	defer func() { clog.InitNoFlag(&buf, clog.Level(0)) }()

	var errBuf bytes.Buffer
	cmd := &initCmd{
		out:     &buf,
		errOut:  &errBuf,
		fs:      afero.NewMemMapFs(),
		client:  kclient,
		crdOnly: true,
		version: "dev",
	}
	err = cmd.run()
	assert.NoError(t, err)

	// WaitForCRDs to be created... the init cmd did NOT wait
	assert.NoError(t, envtest.WaitForCRDs(testenv.Config, crds, envtest.CRDInstallOptions{
		PollInterval: 100 * time.Millisecond,
		MaxTime:      10 * time.Second,
	}))
	defer func() {
		assert.NoError(t, deleteObjects(crds, testClient))
	}()

	// second run will have an output that it already exists
	err = cmd.run()

	assert.Equal(t, "failed to verify installation requirements", err.Error())
	assertStringContains(t, "CRD operators.kudo.dev is already installed. Did you mean to use --upgrade?", errBuf.String())
}

func deleteObjects(objs []runtime.Object, client *testutils.RetryClient) error {
	for _, obj := range objs {
		if err := client.Delete(context.TODO(), obj); err != nil {
			return err
		}
	}

	return testutils.WaitForDelete(client, objs)
}

func deleteInitPrereqs(cmd *initCmd, client *testutils.RetryClient) error {
	opts := kudoinit.NewOptions(cmd.version, cmd.ns, cmd.serviceAccount, cmd.upgrade, cmd.selfSignedWebhookCA)

	objs := append([]runtime.Object{}, prereq.NewWebHookInitializer(opts).Resources()...)
	objs = append(objs, prereq.NewServiceAccountInitializer(opts).Resources()...)
	objs = append(objs, crd.NewInitializer().Resources()...)

	// Namespaced resources aren't waited on after deletion because they aren't GC'ed in this test environment.
	for _, ns := range prereq.NewNamespaceInitializer(opts).Resources() {
		if err := client.Delete(context.TODO(), ns); err != nil {
			return err
		}
	}

	return deleteObjects(objs, client)
}

func getKubeClient(t *testing.T) *kube.Client {
	c, err := kubernetes.NewForConfig(testenv.Config)
	assert.NoError(t, err)
	xc, err := apiextensionsclient.NewForConfig(testenv.Config)
	assert.NoError(t, err)
	cc, err := client.New(testenv.Config, client.Options{})
	assert.NoError(t, err)
	return &kube.Client{KubeClient: c, ExtClient: xc, CtrlClient: cc}
}
