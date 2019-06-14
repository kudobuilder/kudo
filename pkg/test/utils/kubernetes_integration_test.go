// +build integration

package utils

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestCreateOrUpdate(t *testing.T) {
	env := &envtest.Environment{}

	config, err := env.Start()
	assert.Nil(t, err)

	defer env.Stop()

	client, err := client.New(config, client.Options{})
	assert.Nil(t, err)

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

		assert.Nil(t, CreateOrUpdate(context.TODO(), client, SetAnnotation(depToUpdate, "test", "hi"), true))

		quit := make(chan bool)

		go func() {
			for {
				select {
				case <-quit:
					return
				default:
					CreateOrUpdate(context.TODO(), client, SetAnnotation(depToUpdate, "test", fmt.Sprintf("%d", i)), false)
					time.Sleep(time.Millisecond * 75)
				}
			}
		}()

		time.Sleep(time.Millisecond * 50)

		assert.Nil(t, CreateOrUpdate(context.TODO(), client, SetAnnotation(depToUpdate, "test", "hello"), true))

		quit <- true
	}
}

func TestWaitForCRDs(t *testing.T) {
	env := &envtest.Environment{}

	config, err := env.Start()
	assert.Nil(t, err)

	defer env.Stop()

	cl, err := client.New(config, client.Options{
		Scheme: Scheme(),
	})
	assert.Nil(t, err)
	dClient, err := discovery.NewDiscoveryClientForConfig(config)
	assert.Nil(t, err)

	instance := NewResource("kudo.k8s.io/v1alpha1", "Instance", "zk", "ns")

	// Verify that we cannot create the instance, because the test environment is empty.
	assert.IsType(t, &meta.NoKindMatchError{}, cl.Create(context.TODO(), instance))

	// Install all of the CRDs.
	crds, err := InstallManifests(context.TODO(), cl, dClient, "../../../config/crds/")
	assert.Nil(t, err)

	// WaitForCRDs to be created.
	assert.Nil(t, WaitForCRDs(dClient, crds))

	// Kubernetes client caches the types, se we need to re-initialize it.
	cl, err = client.New(config, client.Options{
		Scheme: Scheme(),
	})
	assert.Nil(t, err)

	// make sure that we can create an object of this type now
	assert.Nil(t, cl.Create(context.TODO(), instance))
}
