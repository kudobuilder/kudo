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

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	cmdinit "github.com/kudobuilder/kudo/pkg/kudoctl/cmd/init"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"sigs.k8s.io/yaml"

	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	operatorFileName        = "kudo_v1beta1_operator.yaml"
	operatorVersionFileName = "kudo_v1beta1_operatorversion.yaml"
	instanceFileName        = "kudo_v1beta1_instance.yaml"
	manifestsDir            = "../../../config/crds/"
)

func TestCrds_Config(t *testing.T) {
	crds := cmdinit.CRDs()

	if false {
		// change this to true if you want to one time override the manifests with new values
		// this should be used only when the manifests changed in your PR and you want to update to the newly generated values
		err := writeManifest(operatorFileName, crds.Operator)
		if err != nil {
			t.Errorf("Operator file override failed: %v", err)
		}
		err = writeManifest(operatorVersionFileName, crds.OperatorVersion)
		if err != nil {
			t.Errorf("OperatorVersion file override failed: %v", err)
		}
		if err != nil {
			t.Errorf("Instance file override failed: %v", err)
		}
		err = writeManifest(instanceFileName, crds.Instance)
	}

	assertManifestFileMatch(t, operatorFileName, crds.Operator)
	assertManifestFileMatch(t, operatorVersionFileName, crds.OperatorVersion)
	assertManifestFileMatch(t, instanceFileName, crds.Instance)
}

func writeManifest(fileName string, expectedObject runtime.Object) error {
	expectedContent, err := runtimeObjectAsBytes(expectedObject)
	if err != nil {
		return err
	}

	fmt.Printf("Updating file %s", fileName)
	path := filepath.Join(manifestsDir, fileName)
	if err := ioutil.WriteFile(path, expectedContent, 0644); err != nil {
		return fmt.Errorf("failed to update config file: %s", err)
	}
	return nil
}

func assertManifestFileMatch(t *testing.T, fileName string, expectedObject runtime.Object) {
	expectedContent, err := runtimeObjectAsBytes(expectedObject)
	assert.Nil(t, err)
	path := filepath.Join(manifestsDir, fileName)
	of, err := ioutil.ReadFile(path)
	assert.Nil(t, err)

	assert.Equal(t, string(expectedContent), string(of), "manifest file does not match the existing one")
}

func runtimeObjectAsBytes(o runtime.Object) ([]byte, error) {
	bytes, err := yaml.Marshal(o)
	if err != nil {
		return nil, err
	}
	return bytes, nil
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
	crds := cmdinit.CRDs().AsArray()
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
	crds := cmdinit.CRDs().AsArray()
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
	assert.Equal(t, err.Error(), `error installing: namespace integration-test does not exist - KUDO expects that any namespace except the default kudo-system is created beforehand`)

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
	crds := cmdinit.CRDs().AsArray()
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
	crds := cmdinit.CRDs().AsArray()
	prereqs := cmdinit.Prereq(cmdinit.NewOptions("", ""))
	deleteCRDs(crds, client)
	deletePrereq(prereqs, client)
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
