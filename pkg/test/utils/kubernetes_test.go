package utils

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestNamespaced(t *testing.T) {
	fake := FakeDiscoveryClient()

	for _, test := range []struct {
		testName             string
		resource             runtime.Object
		namespaceShouldBeSet bool
		shouldError          bool
	}{
		{"namespaced resource", NewPod("hello", ""), true, false},
		{"namespace already set", NewPod("hello", "other"), true, false},
		{"not-namespaced resource", NewResource("v1", "Namespace", "hello", ""), false, false},
		{"non-existent resource", NewResource("v1", "Blah", "hello", ""), false, true},
	} {
		t.Run(test.testName, func(t *testing.T) {
			namespace := "world"

			m, _ := meta.Accessor(test.resource)

			actualName, actualNamespace, err := Namespaced(fake, test.resource, namespace)

			if test.shouldError {
				assert.NotNil(t, err)
				assert.Equal(t, "", actualName)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, m.GetName(), actualName)
			}

			if !test.namespaceShouldBeSet {
				namespace = ""
			}

			assert.Equal(t, namespace, actualNamespace)
			assert.Equal(t, namespace, m.GetNamespace())
		})
	}
}

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
