package utils

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

func TestGETAPIResource(t *testing.T) {
	fake := FakeDiscoveryClient()

	apiResource, err := GetAPIResource(fake, schema.GroupVersionKind{
		Kind:    "Pod",
		Version: "v1",
	})
	assert.Nil(t, err)
	assert.Equal(t, apiResource.Kind, "Pod")

	apiResource, err = GetAPIResource(fake, schema.GroupVersionKind{
		Kind:    "NonExistentResourceType",
		Version: "v1",
	})
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "resource type not found")
}

func TestRetry(t *testing.T) {
	index := 0

	assert.Nil(t, Retry(context.TODO(), func(context.Context) error {
		if index == 0 {
			index += 1
			return errors.New("ignore this error")
		}
		index += 1
		return nil
	}, func(err error) bool { return false }, func(err error) bool {
		return err.Error() == "ignore this error"
	}))

	assert.Equal(t, 2, index)
}

func TestRetryWithUnexpectedError(t *testing.T) {
	index := 0

	assert.Equal(t, errors.New("bad error"), Retry(context.TODO(), func(context.Context) error {
		if index == 0 {
			index += 1
			return errors.New("bad error")
		}
		index += 1
		return nil
	}, func(err error) bool { return false }, func(err error) bool {
		return err.Error() == "ignore this error"
	}))
	assert.Equal(t, 1, index)
}

func TestRetryWithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	assert.Equal(t, errors.New("error"), Retry(ctx, func(context.Context) error {
		return errors.New("error")
	}, func(err error) bool { return true }))
}
