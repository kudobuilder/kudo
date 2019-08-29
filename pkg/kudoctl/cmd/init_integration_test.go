// +build integration

package cmd

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"testing"

	cmdinit "github.com/kudobuilder/kudo/pkg/kudoctl/cmd/init"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"

	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
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

func TestIntegInitForCRDs(t *testing.T) {
	// Kubernetes client caches the types, se we need to re-initialize it.
	testClient, err := testutils.NewRetryClient(testenv.Config, client.Options{
		Scheme: testutils.Scheme(),
	})
	assert.Nil(t, err)
	kclient := getKubeClient(t)

	instance := testutils.NewResource("kudo.dev/v1alpha1", "Instance", "zk", "ns")
	// Verify that we cannot create the instance, because the test environment is empty.
	assert.IsType(t, &meta.NoKindMatchError{}, testClient.Create(context.TODO(), instance))

	// Install all of the CRDs.
	crds := cmdinit.CRDs()
	defer deleteCRDs(crds, testClient)

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

func deleteCRDs(crds []runtime.Object, testClient *testutils.RetryClient) {
	for _, crd := range crds {
		testClient.Delete(context.TODO(), crd)
	}
	testutils.WaitForDelete(testClient, crds)
}

func TestIntegInitWithTimeoutsForCRDs(t *testing.T) {
	// Kubernetes client caches the types, se we need to re-initialize it.
	testClient, err := testutils.NewRetryClient(testenv.Config, client.Options{
		Scheme: testutils.Scheme(),
	})
	assert.Nil(t, err)

	kclient := getKubeClient(t)

	instance := testutils.NewResource("kudo.dev/v1alpha1", "Instance", "zk", "ns")
	// Verify that we cannot create the instance, because the test environment is empty.
	assert.IsType(t, &meta.NoKindMatchError{}, testClient.Create(context.TODO(), instance))

	var buf bytes.Buffer
	cmd := &initCmd{
		out:     &buf,
		fs:      afero.NewMemMapFs(),
		timeout: 1,
		wait:    true,
		client:  kclient,
	}
	err = cmd.run()
	assert.NotNil(t, err)

	expected := "watch timed out, readiness uncertain"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func getKubeClient(t *testing.T) *kube.Client {
	c, err := kubernetes.NewForConfig(testenv.Config)
	assert.Nil(t, err)
	xc, err := apiextensionsclient.NewForConfig(testenv.Config)
	assert.Nil(t, err)
	return &kube.Client{KubeClient: c, ExtClient: xc}
}
