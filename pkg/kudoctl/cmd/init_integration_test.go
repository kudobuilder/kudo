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

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/crd"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/prereq"
	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
)

var testenv testutils.TestEnvironment

func TestMain(m *testing.M) {
	var err error

	testenv, err = testutils.StartTestEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	exitCode := m.Run()
	testenv.Environment.Stop()
	os.Exit(exitCode)
}

const (
	operatorFileName        = "kudo.dev_operators.yaml"
	operatorVersionFileName = "kudo.dev_operatorversions.yaml"
	instanceFileName        = "kudo.dev_instances.yaml"
	manifestsDir            = "../../../config/crds/"
)

func TestCrds_Config(t *testing.T) {
	crds := crd.NewInitializer()
	assertManifestFileMatch(t, operatorFileName, crds.Operator)
	assertManifestFileMatch(t, operatorVersionFileName, crds.OperatorVersion)
	assertManifestFileMatch(t, instanceFileName, crds.Instance)
}

func assertManifestFileMatch(t *testing.T, fileName string, expectedObject runtime.Object) {
	expectedContent, err := runtimeObjectAsBytes(expectedObject)
	assert.Nil(t, err)
	path := filepath.Join(manifestsDir, fileName)
	of, err := ioutil.ReadFile(path)
	assert.Nil(t, err)

	assert.Equal(t, string(expectedContent), string(of), fmt.Sprintf("embedded file %s does not match the source, run 'make generate'", fileName))
}

func runtimeObjectAsBytes(o runtime.Object) ([]byte, error) {
	bytes, err := yaml.Marshal(o)
	if err != nil {
		return nil, err
	}
	return append([]byte("\n---\n"), bytes...), nil
}

func TestIntegInitForCRDs(t *testing.T) {
	// Kubernetes client caches the types, se we need to re-initialize it.
	testClient, err := testutils.NewRetryClient(testenv.Config, client.Options{
		Scheme: testutils.Scheme(),
	})
	assert.Nil(t, err)
	kclient := getKubeClient(t)

	instance := testutils.NewResource("kudo.dev/v1beta1", "Instance", "zk", "ns")
	// Verify that we cannot create the instance, because the test environment is empty.
	assert.IsType(t, &meta.NoKindMatchError{}, testClient.Create(context.TODO(), instance))

	// Install all of the CRDs.
	crds := crd.NewInitializer().AsArray()
	defer deleteInitObjects(testClient)

	var buf bytes.Buffer
	cmd := &initCmd{
		out:    &buf,
		fs:     afero.NewMemMapFs(),
		client: kclient,
	}
	err = cmd.run()
	assert.Nil(t, err)

	// WaitForCRDs to be created... the init cmd did NOT wait
	assert.Nil(t, testutils.WaitForCRDs(testenv.DiscoveryClient, crds))

	// Kubernetes client caches the types, se we need to re-initialize it.
	testClient, err = testutils.NewRetryClient(testenv.Config, client.Options{
		Scheme: testutils.Scheme(),
	})
	assert.Nil(t, err)

	// make sure that we can create an object of this type now
	assert.Nil(t, testClient.Create(context.TODO(), instance))
}

func TestIntegInitWithNameSpace(t *testing.T) {
	namespace := "integration-test"
	// Kubernetes client caches the types, se we need to re-initialize it.
	testClient, err := testutils.NewRetryClient(testenv.Config, client.Options{
		Scheme: testutils.Scheme(),
	})
	assert.Nil(t, err)
	kclient := getKubeClient(t)

	instance := testutils.NewResource("kudo.dev/v1beta1", "Instance", "zk", "ns")
	// Verify that we cannot create the instance, because the test environment is empty.
	assert.IsType(t, &meta.NoKindMatchError{}, testClient.Create(context.TODO(), instance))

	// Install all of the CRDs.
	crds := crd.NewInitializer().AsArray()
	defer deleteInitObjects(testClient)

	var buf bytes.Buffer
	cmd := &initCmd{
		out:    &buf,
		fs:     afero.NewMemMapFs(),
		client: kclient,
		ns:     namespace,
	}

	// On first attempt, the namespace does not exist, so the error is expected.
	err = cmd.run()
	require.Error(t, err)
	assert.Equal(t, err.Error(), `error installing: prerequisites: failed to install: namespace integration-test does not exist - KUDO expects that any namespace except the default kudo-system is created beforehand`)

	// Then we manually create the namespace.
	ns := testutils.NewResource("v1", "Namespace", namespace, "")
	assert.NoError(t, testClient.Create(context.TODO(), ns))
	defer testClient.Delete(context.TODO(), ns)

	// On second attempt run should succeed.
	err = cmd.run()
	assert.NoError(t, err)

	// WaitForCRDs to be created... the init cmd did NOT wait
	assert.Nil(t, testutils.WaitForCRDs(testenv.DiscoveryClient, crds))

	// Kubernetes client caches the types, so we need to re-initialize it.
	testClient, err = testutils.NewRetryClient(testenv.Config, client.Options{
		Scheme: testutils.Scheme(),
	})
	assert.Nil(t, err)
	kclient = getKubeClient(t)

	// make sure that the controller lives in the correct namespace
	statefulsets, err := kclient.KubeClient.AppsV1().StatefulSets(namespace).List(metav1.ListOptions{})
	assert.Nil(t, err)

	kudoControllerFound := false
	for _, ss := range statefulsets.Items {
		if ss.Name == "kudo-controller-manager" {
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

func TestIntegInitWithServiceAccount(t *testing.T) {
	namespace := "sa-integration-test"
	serviceAccount := "sa-integration"
	// Kubernetes client caches the types, se we need to re-initialize it.
	testClient, err := testutils.NewRetryClient(testenv.Config, client.Options{
		Scheme: testutils.Scheme(),
	})
	assert.Nil(t, err)
	kclient := getKubeClient(t)

	instance := testutils.NewResource("kudo.dev/v1beta1", "Instance", "zk", "ns")
	// Verify that we cannot create the instance, because the test environment is empty.
	assert.IsType(t, &meta.NoKindMatchError{}, testClient.Create(context.TODO(), instance))

	// Install all of the CRDs.
	crds := crd.NewInitializer().AsArray()
	defer deleteInitObjects(testClient)

	var buf bytes.Buffer
	cmd := &initCmd{
		out:            &buf,
		fs:             afero.NewMemMapFs(),
		client:         kclient,
		ns:             namespace,
		serviceAccount: "test-account",
	}

	// Manually create the namespace and the serviceAccount to be used later
	ns := testutils.NewResource("v1", "Namespace", namespace, "")
	assert.NoError(t, testClient.Create(context.TODO(), ns))
	defer testClient.Delete(context.TODO(), ns)
	sa := testutils.NewResource("v1", "ServiceAccount", serviceAccount, namespace)
	assert.NoError(t, testClient.Create(context.TODO(), sa))
	defer testClient.Delete(context.TODO(), sa)

	// Test Case 1, the serviceAccount does not exist, expect serviceAccount not exists error
	err = cmd.run()
	require.Error(t, err)
	assert.Equal(t, `error installing: prerequisites: failed to install: Service Account test-account does not exists - KUDO expects the serviceAccount to be present in the namespace sa-integration-test`, err.Error())

	// Create the serviceAccount, in the default namespace.
	ns2 := testutils.NewResource("v1", "Namespace", "test-ns", "")
	assert.NoError(t, testClient.Create(context.TODO(), ns2))
	defer testClient.Delete(context.TODO(), ns2)
	sa2 := testutils.NewResource("v1", "ServiceAccount", "sa-nonadmin", "test-ns")
	assert.NoError(t, testClient.Create(context.TODO(), sa2))
	defer testClient.Delete(context.TODO(), sa2)

	// Test Case 2, the serviceAccount exists, but does not part of clusterrolebindings
	cmd.serviceAccount = "sa-nonadmin"
	cmd.ns = "test-ns"
	err = cmd.run()
	require.Error(t, err)
	assert.Equal(t, `error installing: prerequisites: failed to install: Service Account sa-nonadmin does not have cluster-admin role - KUDO expects the serviceAccount passed to be in the namespace test-ns and to have cluster-admin role`, err.Error())

	// Test case 3: Run Init command with a serviceAccount that does not have cluster-admin role.
	cmd.serviceAccount = serviceAccount
	cmd.ns = namespace
	crb := testutils.NewClusterRoleBinding("rbac.authorization.k8s.io/v1", "ClusterRoleBinding", "kudo-test1", "test-ns", serviceAccount, "cluster-temp")
	assert.NoError(t, testClient.Create(context.TODO(), crb))
	defer testClient.Delete(context.TODO(), crb)

	err = cmd.run()
	require.Error(t, err)
	assert.Equal(t, `error installing: prerequisites: failed to install: Service Account sa-integration does not have cluster-admin role - KUDO expects the serviceAccount passed to be in the namespace sa-integration-test and to have cluster-admin role`, err.Error())

	// Test case 4: Run Init command with a serviceAccount that does not have cluster-admin role.
	crb2 := testutils.NewClusterRoleBinding("rbac.authorization.k8s.io/v1", "ClusterRoleBinding", "kudo-test2", namespace, serviceAccount, "cluster-temp")
	assert.NoError(t, testClient.Create(context.TODO(), crb2))
	defer testClient.Delete(context.TODO(), crb2)

	err = cmd.run()
	require.Error(t, err)
	assert.Equal(t, `error installing: prerequisites: failed to install: Service Account sa-integration does not have cluster-admin role - KUDO expects the serviceAccount passed to be in the namespace sa-integration-test and to have cluster-admin role`, err.Error())

	// Test case 5: Run Init command with a serviceAccount that is present in the cluster and also has cluster-admin role.
	crb3 := testutils.NewClusterRoleBinding("rbac.authorization.k8s.io/v1", "ClusterRoleBinding", "kudo-clusterrole-binding", namespace, serviceAccount, "cluster-admin")
	assert.NoError(t, testClient.Create(context.TODO(), crb3))
	defer testClient.Delete(context.TODO(), crb3)

	err = cmd.run()
	assert.NoError(t, err)

	// WaitForCRDs to be created... the init cmd did NOT wait
	assert.Nil(t, testutils.WaitForCRDs(testenv.DiscoveryClient, crds))

	// Kubernetes client caches the types, so we need to re-initialize it.
	testClient, err = testutils.NewRetryClient(testenv.Config, client.Options{
		Scheme: testutils.Scheme(),
	})
	assert.Nil(t, err)
	kclient = getKubeClient(t)

	// make sure that the controller lives in the correct namespace
	statefulsets, err := kclient.KubeClient.AppsV1().StatefulSets(namespace).List(metav1.ListOptions{})
	assert.Nil(t, err)

	kudoControllerFound := false
	for _, ss := range statefulsets.Items {
		if ss.Name == "kudo-controller-manager" {
			kudoControllerFound = true
		}
	}
	assert.True(t, kudoControllerFound, fmt.Sprintf("No kudo-controller-manager statefulset found in namespace %s", namespace))
}

func TestNoErrorOnReInit(t *testing.T) {
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
	crds := crd.NewInitializer().AsArray()
	defer deleteInitObjects(testClient)

	var buf bytes.Buffer
	clog.InitNoFlag(&buf, clog.Level(4))
	defer func() { clog.InitNoFlag(&buf, clog.Level(0)) }()

	cmd := &initCmd{
		out:     &buf,
		fs:      afero.NewMemMapFs(),
		client:  kclient,
		crdOnly: true,
	}
	err = cmd.run()
	assert.Nil(t, err)

	// WaitForCRDs to be created... the init cmd did NOT wait
	assert.Nil(t, testutils.WaitForCRDs(testenv.DiscoveryClient, crds))

	//	 if the CRD exists and we init again there should be no error
	testClient, err = testutils.NewRetryClient(testenv.Config, client.Options{
		Scheme: testutils.Scheme(),
	})
	assert.Nil(t, err)
	kclient = getKubeClient(t)

	// second run will have an output that it already exists
	err = cmd.run()
	assert.Nil(t, err)
	assert.True(t, strings.Contains(buf.String(), "crd operators.kudo.dev already exists"))
}

func deleteInitObjects(client *testutils.RetryClient) {
	crds := crd.NewInitializer()
	prereqs := prereq.NewInitializer(kudoinit.NewOptions("", "", "", []string{}))
	deleteCRDs(crds.AsArray(), client)
	deletePrereq(prereqs.AsArray(), client)
}

func deleteCRDs(crds []runtime.Object, client *testutils.RetryClient) {

	for _, crd := range crds {
		client.Delete(context.TODO(), crd)
	}
	testutils.WaitForDelete(client, crds)
}

func deletePrereq(prereqs []runtime.Object, client *testutils.RetryClient) {
	for _, prereq := range prereqs {
		client.Delete(context.TODO(), prereq)
	}
	testutils.WaitForDelete(client, prereqs)
}

func getKubeClient(t *testing.T) *kube.Client {
	c, err := kubernetes.NewForConfig(testenv.Config)
	assert.Nil(t, err)
	xc, err := apiextensionsclient.NewForConfig(testenv.Config)
	assert.Nil(t, err)
	return &kube.Client{KubeClient: c, ExtClient: xc}
}
