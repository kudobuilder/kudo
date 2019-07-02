package test

import (
	"context"
	"testing"
	"time"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Verify the test state as loaded from disk.
// Each test provides a path to a set of test steps and their rendered result.
func TestStepClean(t *testing.T) {
	namespace := "world"

	pod := testutils.NewPod("hello", "")

	podWithNamespace := testutils.WithNamespace(pod, namespace)
	pod2WithNamespace := testutils.NewPod("hello2", namespace)
	pod2WithDiffNamespace := testutils.NewPod("hello2", "different-namespace")

	step := Step{
		Apply: []runtime.Object{
			pod.DeepCopyObject(), pod2WithDiffNamespace.DeepCopyObject(), testutils.NewPod("does-not-exist", ""),
		},
		Client:          fake.NewFakeClient(pod, pod2WithNamespace, pod2WithDiffNamespace),
		DiscoveryClient: testutils.FakeDiscoveryClient(),
	}

	assert.Nil(t, step.Clean(namespace))

	assert.True(t, k8serrors.IsNotFound(step.Client.Get(context.TODO(), testutils.ObjectKey(podWithNamespace), podWithNamespace)))
	assert.True(t, k8serrors.IsNotFound(step.Client.Get(context.TODO(), testutils.ObjectKey(pod2WithNamespace), pod2WithNamespace)))
	assert.Nil(t, step.Client.Get(context.TODO(), testutils.ObjectKey(pod2WithDiffNamespace), pod2WithDiffNamespace))
}

// Verify the test state as loaded from disk.
// Each test provides a path to a set of test steps and their rendered result.
func TestStepCreate(t *testing.T) {
	namespace := "world"

	pod := testutils.NewPod("hello", "")
	podWithNamespace := testutils.NewPod("hello2", "different-namespace")
	clusterScopedResource := testutils.NewResource("v1", "Namespace", "my-namespace", "")
	podToUpdate := testutils.NewPod("update-me", "")
	updateToApply := testutils.WithSpec(podToUpdate, map[string]interface{}{
		"replicas": 2,
	})

	step := Step{
		Logger: testutils.NewTestLogger(t, ""),
		Apply: []runtime.Object{
			pod.DeepCopyObject(), podWithNamespace.DeepCopyObject(), clusterScopedResource, updateToApply,
		},
		Client:          fake.NewFakeClient(testutils.WithNamespace(podToUpdate, "world")),
		DiscoveryClient: testutils.FakeDiscoveryClient(),
	}

	assert.Equal(t, []error{}, step.Create(namespace))

	assert.Nil(t, step.Client.Get(context.TODO(), testutils.ObjectKey(pod), pod))
	assert.Nil(t, step.Client.Get(context.TODO(), testutils.ObjectKey(clusterScopedResource), clusterScopedResource))

	assert.Nil(t, step.Client.Get(context.TODO(), testutils.ObjectKey(podToUpdate), podToUpdate))
	assert.Equal(t, updateToApply, podToUpdate)

	assert.True(t, k8serrors.IsNotFound(step.Client.Get(context.TODO(), testutils.ObjectKey(podWithNamespace), podWithNamespace)))
	actual := testutils.NewPod("hello2", namespace)
	assert.Nil(t, step.Client.Get(context.TODO(), testutils.ObjectKey(actual), actual))
}

// Verify the test state as loaded from disk.
// Each test provides a path to a set of test steps and their rendered result.
func TestStepDeleteExisting(t *testing.T) {
	namespace := "world"

	podToDelete := testutils.NewPod("delete-me", "world")
	podToDeleteDefaultNS := testutils.NewPod("also-delete-me", "default")
	podToKeep := testutils.NewPod("keep-me", "world")

	step := Step{
		Logger: testutils.NewTestLogger(t, ""),
		Step: &kudo.TestStep{
			Delete: []corev1.ObjectReference{
				{
					Kind:       "Pod",
					APIVersion: "v1",
					Name:       "delete-me",
				},
				{
					Kind:       "Pod",
					APIVersion: "v1",
					Name:       "also-delete-me",
					Namespace:  "default",
				},
			},
		},
		Client:          fake.NewFakeClient(podToDelete, podToKeep, podToDeleteDefaultNS),
		DiscoveryClient: testutils.FakeDiscoveryClient(),
	}

	assert.Nil(t, step.Client.Get(context.TODO(), testutils.ObjectKey(podToKeep), podToKeep))
	assert.Nil(t, step.Client.Get(context.TODO(), testutils.ObjectKey(podToDelete), podToDelete))
	assert.Nil(t, step.Client.Get(context.TODO(), testutils.ObjectKey(podToDeleteDefaultNS), podToDeleteDefaultNS))

	assert.Nil(t, step.DeleteExisting(namespace))

	assert.Nil(t, step.Client.Get(context.TODO(), testutils.ObjectKey(podToKeep), podToKeep))
	assert.True(t, k8serrors.IsNotFound(step.Client.Get(context.TODO(), testutils.ObjectKey(podToDelete), podToDelete)))
	assert.True(t, k8serrors.IsNotFound(step.Client.Get(context.TODO(), testutils.ObjectKey(podToDeleteDefaultNS), podToDeleteDefaultNS)))
}

func TestCheckResource(t *testing.T) {
	for _, test := range []struct {
		testName    string
		actual      runtime.Object
		expected    runtime.Object
		shouldError bool
	}{
		{
			testName: "resource matches",
			actual:   testutils.NewPod("hello", ""),
			expected: testutils.NewPod("hello", ""),
		},
		{
			testName:    "resource mis-match",
			actual:      testutils.NewPod("hello", ""),
			expected:    testutils.WithSpec(testutils.NewPod("hello", ""), map[string]interface{}{"invalid": "key"}),
			shouldError: true,
		},
		{
			testName: "resource subset match",
			actual: testutils.WithSpec(testutils.NewPod("hello", ""), map[string]interface{}{
				"ignored": "key",
				"seen":    "key",
			}),
			expected: testutils.WithSpec(testutils.NewPod("hello", ""), map[string]interface{}{
				"seen": "key",
			}),
		},
		{
			testName:    "resource does not exist",
			actual:      testutils.NewPod("other", ""),
			expected:    testutils.NewPod("hello", ""),
			shouldError: true,
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			fakeDiscovery := testutils.FakeDiscoveryClient()
			namespace := "world"

			_, _, err := testutils.Namespaced(fakeDiscovery, test.actual, namespace)
			assert.Nil(t, err)

			step := Step{
				Logger:          testutils.NewTestLogger(t, ""),
				Client:          fake.NewFakeClient(test.actual),
				DiscoveryClient: fakeDiscovery,
			}

			errors := step.CheckResource(test.expected, namespace)

			if test.shouldError {
				assert.NotEqual(t, []error{}, errors)
			} else {
				assert.Equal(t, []error{}, errors)
			}
		})
	}
}

func TestRun(t *testing.T) {
	for _, test := range []struct {
		testName     string
		shouldError  bool
		Step         Step
		updateMethod func(*testing.T, client.Client)
	}{
		{
			"successful run", false, Step{
				Apply: []runtime.Object{
					testutils.NewPod("hello", ""),
				},
				Asserts: []runtime.Object{
					testutils.NewPod("hello", ""),
				},
			}, nil,
		},
		{
			"failed run", true, Step{
				Apply: []runtime.Object{
					testutils.NewPod("hello", ""),
				},
				Asserts: []runtime.Object{
					testutils.WithStatus(testutils.NewPod("hello", ""), map[string]interface{}{
						"phase": "Ready",
					}),
				},
			}, nil,
		},
		{
			"delayed run", false, Step{
				Apply: []runtime.Object{
					testutils.NewPod("hello", ""),
				},
				Asserts: []runtime.Object{
					testutils.WithStatus(testutils.NewPod("hello", ""), map[string]interface{}{
						"phase": "Ready",
					}),
				},
			}, func(t *testing.T, client client.Client) {
				// mock kubelet to set the pod status
				assert.Nil(t, client.Update(context.TODO(), testutils.WithStatus(testutils.NewPod("hello", "world"), map[string]interface{}{
					"phase": "Ready",
				})))
			},
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			test.Step.Assert = &kudo.TestAssert{
				Timeout: 1,
			}

			test.Step.Client = fake.NewFakeClient()
			test.Step.DiscoveryClient = testutils.FakeDiscoveryClient()
			test.Step.Logger = testutils.NewTestLogger(t, "")

			if test.updateMethod != nil {
				test.Step.Assert.Timeout = 10

				go func() {
					time.Sleep(time.Second * 2)
					test.updateMethod(t, test.Step.Client)
				}()
			}

			errors := test.Step.Run("world")

			if test.shouldError {
				assert.NotEqual(t, []error{}, errors)
			} else {
				assert.Equal(t, []error{}, errors)
			}
		})
	}
}
