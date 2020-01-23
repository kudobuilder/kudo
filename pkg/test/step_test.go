package test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	harness "github.com/kudobuilder/kudo/pkg/apis/testharness/v1beta1"
	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
)

const (
	testNamespace = "world"
)

// Verify the test state as loaded from disk.
// Each test provides a path to a set of test steps and their rendered result.
func TestStepClean(t *testing.T) {

	pod := testutils.NewPod("hello", "")

	podWithNamespace := testutils.WithNamespace(pod, testNamespace)
	pod2WithNamespace := testutils.NewPod("hello2", testNamespace)
	pod2WithDiffNamespace := testutils.NewPod("hello2", "different-namespace")

	cl := fake.NewFakeClient(pod, pod2WithNamespace, pod2WithDiffNamespace)

	step := Step{
		Apply: []runtime.Object{
			pod.DeepCopyObject(), pod2WithDiffNamespace.DeepCopyObject(), testutils.NewPod("does-not-exist", ""),
		},
		Client:          func(bool) (client.Client, error) { return cl, nil },
		DiscoveryClient: func() (discovery.DiscoveryInterface, error) { return testutils.FakeDiscoveryClient(), nil },
	}

	assert.Nil(t, step.Clean(testNamespace))

	assert.True(t, k8serrors.IsNotFound(cl.Get(context.TODO(), testutils.ObjectKey(podWithNamespace), podWithNamespace)))
	assert.Nil(t, cl.Get(context.TODO(), testutils.ObjectKey(pod2WithNamespace), pod2WithNamespace))
	assert.True(t, k8serrors.IsNotFound(cl.Get(context.TODO(), testutils.ObjectKey(pod2WithDiffNamespace), pod2WithDiffNamespace)))
}

// Verify the test state as loaded from disk.
// Each test provides a path to a set of test steps and their rendered result.
func TestStepCreate(t *testing.T) {

	pod := testutils.NewPod("hello", "")
	podWithNamespace := testutils.NewPod("hello2", "different-namespace")
	clusterScopedResource := testutils.NewResource("v1", "Namespace", "my-namespace", "")
	podToUpdate := testutils.NewPod("update-me", "")
	specToApply := map[string]interface{}{
		"replicas": int64(2),
	}
	updateToApply := testutils.WithSpec(t, podToUpdate, specToApply)

	cl := fake.NewFakeClient(testutils.WithNamespace(podToUpdate, testNamespace))

	step := Step{
		Logger: testutils.NewTestLogger(t, ""),
		Apply: []runtime.Object{
			pod.DeepCopyObject(), podWithNamespace.DeepCopyObject(), clusterScopedResource, updateToApply,
		},
		Client:          func(bool) (client.Client, error) { return cl, nil },
		DiscoveryClient: func() (discovery.DiscoveryInterface, error) { return testutils.FakeDiscoveryClient(), nil },
	}

	assert.Equal(t, []error{}, step.Create(testNamespace))

	assert.Nil(t, cl.Get(context.TODO(), testutils.ObjectKey(pod), pod))
	assert.Nil(t, cl.Get(context.TODO(), testutils.ObjectKey(clusterScopedResource), clusterScopedResource))

	updatedPod := &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Pod"}}
	assert.Nil(t, cl.Get(context.TODO(), testutils.ObjectKey(podToUpdate), updatedPod))
	assert.Equal(t, specToApply, updatedPod.Object["spec"])

	assert.Nil(t, cl.Get(context.TODO(), testutils.ObjectKey(podWithNamespace), podWithNamespace))
	actual := testutils.NewPod("hello2", testNamespace)
	assert.True(t, k8serrors.IsNotFound(cl.Get(context.TODO(), testutils.ObjectKey(actual), actual)))
}

// Verify that the DeleteExisting method properly cleans up resources during a test step.
func TestStepDeleteExisting(t *testing.T) {

	podToDelete := testutils.NewPod("delete-me", testNamespace)
	podToDeleteDefaultNS := testutils.NewPod("also-delete-me", "default")
	podToKeep := testutils.NewPod("keep-me", testNamespace)

	cl := fake.NewFakeClient(podToDelete, podToKeep, podToDeleteDefaultNS)

	step := Step{
		Logger: testutils.NewTestLogger(t, ""),
		Step: &harness.TestStep{
			Delete: []harness.ObjectReference{
				{
					ObjectReference: corev1.ObjectReference{
						Kind:       "Pod",
						APIVersion: "v1",
						Name:       "delete-me",
					},
				},
				{
					ObjectReference: corev1.ObjectReference{
						Kind:       "Pod",
						APIVersion: "v1",
						Name:       "also-delete-me",
						Namespace:  "default",
					},
				},
			},
		},
		Client:          func(bool) (client.Client, error) { return cl, nil },
		DiscoveryClient: func() (discovery.DiscoveryInterface, error) { return testutils.FakeDiscoveryClient(), nil },
	}

	assert.Nil(t, cl.Get(context.TODO(), testutils.ObjectKey(podToKeep), podToKeep))
	assert.Nil(t, cl.Get(context.TODO(), testutils.ObjectKey(podToDelete), podToDelete))
	assert.Nil(t, cl.Get(context.TODO(), testutils.ObjectKey(podToDeleteDefaultNS), podToDeleteDefaultNS))

	assert.Nil(t, step.DeleteExisting(testNamespace))

	assert.Nil(t, cl.Get(context.TODO(), testutils.ObjectKey(podToKeep), podToKeep))
	assert.True(t, k8serrors.IsNotFound(cl.Get(context.TODO(), testutils.ObjectKey(podToDelete), podToDelete)))
	assert.True(t, k8serrors.IsNotFound(cl.Get(context.TODO(), testutils.ObjectKey(podToDeleteDefaultNS), podToDeleteDefaultNS)))
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
			expected:    testutils.WithSpec(t, testutils.NewPod("hello", ""), map[string]interface{}{"invalid": "key"}),
			shouldError: true,
		},
		{
			testName: "resource subset match",
			actual: testutils.WithSpec(t, testutils.NewPod("hello", ""), map[string]interface{}{
				"ignored": "key",
				"seen":    "key",
			}),
			expected: testutils.WithSpec(t, testutils.NewPod("hello", ""), map[string]interface{}{
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
			namespace := testNamespace

			_, _, err := testutils.Namespaced(fakeDiscovery, test.actual, namespace)
			assert.Nil(t, err)

			step := Step{
				Logger:          testutils.NewTestLogger(t, ""),
				Client:          func(bool) (client.Client, error) { return fake.NewFakeClient(test.actual), nil },
				DiscoveryClient: func() (discovery.DiscoveryInterface, error) { return fakeDiscovery, nil },
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

func TestCheckResourceAbsent(t *testing.T) {
	for _, test := range []struct {
		name        string
		actual      runtime.Object
		expected    runtime.Object
		shouldError bool
	}{
		{
			name:        "resource matches",
			actual:      testutils.NewPod("hello", ""),
			expected:    testutils.NewPod("hello", ""),
			shouldError: true,
		},
		{
			name:     "resource mis-match",
			actual:   testutils.NewPod("hello", ""),
			expected: testutils.WithSpec(t, testutils.NewPod("hello", ""), map[string]interface{}{"invalid": "key"}),
		},
		{
			name:     "resource does not exist",
			actual:   testutils.NewPod("other", ""),
			expected: testutils.NewPod("hello", ""),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fakeDiscovery := testutils.FakeDiscoveryClient()

			_, _, err := testutils.Namespaced(fakeDiscovery, test.actual, testNamespace)
			assert.Nil(t, err)

			step := Step{
				Logger:          testutils.NewTestLogger(t, ""),
				Client:          func(bool) (client.Client, error) { return fake.NewFakeClient(test.actual), nil },
				DiscoveryClient: func() (discovery.DiscoveryInterface, error) { return fakeDiscovery, nil },
			}

			error := step.CheckResourceAbsent(test.expected, testNamespace)

			if test.shouldError {
				assert.NotNil(t, error)
			} else {
				assert.Nil(t, error)
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
					testutils.WithStatus(t, testutils.NewPod("hello", ""), map[string]interface{}{
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
					testutils.WithStatus(t, testutils.NewPod("hello", ""), map[string]interface{}{
						"phase": "Ready",
					}),
				},
			}, func(t *testing.T, client client.Client) {
				// mock kubelet to set the pod status
				assert.Nil(t, client.Update(context.TODO(), testutils.WithStatus(t, testutils.NewPod("hello", testNamespace), map[string]interface{}{
					"phase": "Ready",
				})))
			},
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			test.Step.Assert = &harness.TestAssert{
				Timeout: 1,
			}

			cl := fake.NewFakeClient()

			test.Step.Client = func(bool) (client.Client, error) { return cl, nil }
			test.Step.DiscoveryClient = func() (discovery.DiscoveryInterface, error) { return testutils.FakeDiscoveryClient(), nil }
			test.Step.Logger = testutils.NewTestLogger(t, "")

			if test.updateMethod != nil {
				test.Step.Assert.Timeout = 10

				go func() {
					time.Sleep(time.Second * 2)
					test.updateMethod(t, cl)
				}()
			}

			errors := test.Step.Run(testNamespace)

			if test.shouldError {
				assert.NotEqual(t, []error{}, errors)
			} else {
				assert.Equal(t, []error{}, errors)
			}
		})
	}
}
