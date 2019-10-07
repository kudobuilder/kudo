// +build integration

package cmd

import (
	"bytes"
	"context"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
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

func TestNoErrorOnReInit(t *testing.T) {
	//	 if the CRD exists and we init again there should be no error
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
	crds := cmdinit.CRDs()
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
