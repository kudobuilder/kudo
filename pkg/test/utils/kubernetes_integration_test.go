// +build integration

package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var testenv TestEnvironment

func TestMain(m *testing.M) {
	var err error

	testenv, err = StartTestEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	exitCode := m.Run()
	testenv.Environment.Stop()
	os.Exit(exitCode)
}

func TestCreateOrUpdate(t *testing.T) {
	// Run the test a bunch of times to try to trigger a conflict and ensure that it handles conflicts properly.
	for i := 0; i < 10; i++ {
		depToUpdate := WithSpec(NewPod("update-me", fmt.Sprintf("default-%d", i)), map[string]interface{}{
			"containers": []map[string]interface{}{
				{
					"image": "nginx",
					"name":  "nginx",
				},
			},
		})

		_, err := CreateOrUpdate(context.TODO(), testenv.Client, SetAnnotation(depToUpdate, "test", "hi"), true)
		assert.Nil(t, err)

		quit := make(chan bool)

		go func() {
			for {
				select {
				case <-quit:
					return
				default:
					CreateOrUpdate(context.TODO(), testenv.Client, SetAnnotation(depToUpdate, "test", fmt.Sprintf("%d", i)), false)
					time.Sleep(time.Millisecond * 75)
				}
			}
		}()

		time.Sleep(time.Millisecond * 50)

		_, err = CreateOrUpdate(context.TODO(), testenv.Client, SetAnnotation(depToUpdate, "test", "hello"), true)
		assert.Nil(t, err)

		quit <- true
	}
}

func TestWaitForCRDs(t *testing.T) {
	// Kubernetes client caches the types, se we need to re-initialize it.
	testClient, err := NewRetryClient(testenv.Config, client.Options{
		Scheme: Scheme(),
	})
	assert.Nil(t, err)

	instance := NewResource("kudo.k8s.io/v1alpha1", "Instance", "zk", "ns")

	// Verify that we cannot create the instance, because the test environment is empty.
	assert.IsType(t, &meta.NoKindMatchError{}, testClient.Create(context.TODO(), instance))

	// Install all of the CRDs.
	crds, err := InstallManifests(context.TODO(), testClient, testenv.DiscoveryClient, "../../../config/crds/", NewResource("apiextensions.k8s.io/v1beta1", "CustomResourceDefinition", "", ""))
	assert.Nil(t, err)

	// WaitForCRDs to be created.
	assert.Nil(t, WaitForCRDs(testenv.DiscoveryClient, crds))

	// Kubernetes client caches the types, se we need to re-initialize it.
	testClient, err = NewRetryClient(testenv.Config, client.Options{
		Scheme: Scheme(),
	})
	assert.Nil(t, err)

	// make sure that we can create an object of this type now
	assert.Nil(t, testClient.Create(context.TODO(), instance))
}
