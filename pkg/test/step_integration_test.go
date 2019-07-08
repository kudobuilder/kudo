// +build integration

package test

import (
	"context"
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestCheckResourceIntegration(t *testing.T) {
	env := &envtest.Environment{}

	config, err := env.Start()
	assert.Nil(t, err)

	defer env.Stop()

	cl, err := client.New(config, client.Options{
		Scheme: testutils.Scheme(),
	})
	assert.Nil(t, err)
	dClient, err := discovery.NewDiscoveryClientForConfig(config)
	assert.Nil(t, err)

	for _, test := range []struct {
		testName    string
		actual      []runtime.Object
		expected    runtime.Object
		shouldError bool
	}{
		{
			testName: "match object by labels, first in list matches",
			actual: []runtime.Object{
				testutils.WithSpec(testutils.WithLabels(testutils.NewPod("aa", ""), map[string]string{
					"app": "nginx",
				}), map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"image": "nginx:1.7.9",
							"name":  "nginx",
						},
					},
				}),
				testutils.WithSpec(testutils.WithLabels(testutils.NewPod("bb", ""), map[string]string{
					"app": "not-match",
				}), map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"image": "nginx:1.7.9",
							"name":  "nginx",
						},
					},
				}),
			},
			expected: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "nginx",
						},
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"image": "nginx:1.7.9",
								"name":  "nginx",
							},
						},
					},
				},
			},
		},
		{
			testName: "match object by labels, last in list matches",
			actual: []runtime.Object{
				testutils.WithSpec(testutils.WithLabels(testutils.NewPod("aa", ""), map[string]string{
					"app": "not-match",
				}), map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"image": "nginx:1.7.9",
							"name":  "nginx",
						},
					},
				}),
				testutils.WithSpec(testutils.WithLabels(testutils.NewPod("bb", ""), map[string]string{
					"app": "nginx",
				}), map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"image": "nginx:1.7.9",
							"name":  "nginx",
						},
					},
				}),
			},
			expected: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "nginx",
						},
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"image": "nginx:1.7.9",
								"name":  "nginx",
							},
						},
					},
				},
			},
		},
		{
			testName: "match object by labels, does not exist",
			actual: []runtime.Object{
				testutils.WithSpec(testutils.WithLabels(testutils.NewPod("hello", ""), map[string]string{
					"app": "NOT-A-MATCH",
				}), map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"image": "nginx:1.7.9",
							"name":  "nginx",
						},
					},
				}),
			},
			expected: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "nginx",
						},
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"image": "nginx:1.7.9",
								"name":  "nginx",
							},
						},
					},
				},
			},
			shouldError: true,
		},
		{
			testName: "match object by labels, field mismatch",
			actual: []runtime.Object{
				testutils.WithSpec(testutils.WithLabels(testutils.NewPod("hello", ""), map[string]string{
					"app": "nginx",
				}), map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"image": "otherimage:latest",
							"name":  "nginx",
						},
					},
				}),
			},
			expected: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "nginx",
						},
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"image": "nginx:1.7.9",
								"name":  "nginx",
							},
						},
					},
				},
			},
			shouldError: true,
		},
		{
			testName: "step should fail if there are no objects of the same type in the namespace",
			actual:   []runtime.Object{},
			expected: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Pod",
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "nginx",
						},
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"image": "nginx:1.7.9",
								"name":  "nginx",
							},
						},
					},
				},
			},
			shouldError: true,
		},
	} {
		t.Run(test.testName, func(t *testing.T) {
			namespace := fmt.Sprintf("kudo-test-%s", petname.Generate(2, "-"))

			assert.Nil(t, cl.Create(context.TODO(), testutils.NewResource("v1", "Namespace", namespace, "")))

			for _, actual := range test.actual {
				_, _, err := testutils.Namespaced(dClient, actual, namespace)
				assert.Nil(t, err)

				assert.Nil(t, cl.Create(context.TODO(), actual))
			}

			step := Step{
				Logger:          testutils.NewTestLogger(t, ""),
				Client:          cl,
				DiscoveryClient: dClient,
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

// Verify that the DeleteExisting method properly cleans up resources that are matched on labels during a test step.
func TestStepDeleteExistingLabelMatch(t *testing.T) {
	env := &envtest.Environment{}

	config, err := env.Start()
	assert.Nil(t, err)

	defer env.Stop()

	cl, err := client.New(config, client.Options{
		Scheme: testutils.Scheme(),
	})
	assert.Nil(t, err)
	dClient, err := discovery.NewDiscoveryClientForConfig(config)
	assert.Nil(t, err)

	namespace := "world"

	podSpec := map[string]interface{}{
		"containers": []interface{}{
			map[string]interface{}{
				"image": "otherimage:latest",
				"name":  "nginx",
			},
		},
	}

	podToDelete := testutils.WithSpec(testutils.WithLabels(testutils.NewPod("aa-delete-me", "world"), map[string]string{
		"hello": "world",
	}), podSpec)

	podToKeep := testutils.WithSpec(testutils.WithLabels(testutils.NewPod("bb-dont-delete-me", "world"), map[string]string{
		"bye": "moon",
	}), podSpec)

	podToDelete2 := testutils.WithSpec(testutils.WithLabels(testutils.NewPod("cc-delete-me", "world"), map[string]string{
		"hello": "world",
	}), podSpec)

	step := Step{
		Logger: testutils.NewTestLogger(t, ""),
		Step: &kudo.TestStep{
			Delete: []kudo.ObjectReference{
				{
					ObjectReference: corev1.ObjectReference{
						Kind:       "Pod",
						APIVersion: "v1",
					},
					Labels: map[string]string{
						"hello": "world",
					},
				},
			},
		},
		Client:          cl,
		DiscoveryClient: dClient,
	}

	assert.Nil(t, step.Client.Create(context.TODO(), podToKeep))
	assert.Nil(t, step.Client.Create(context.TODO(), podToDelete))
	assert.Nil(t, step.Client.Create(context.TODO(), podToDelete2))

	assert.Nil(t, step.Client.Get(context.TODO(), testutils.ObjectKey(podToKeep), podToKeep))
	assert.Nil(t, step.Client.Get(context.TODO(), testutils.ObjectKey(podToDelete), podToDelete))
	assert.Nil(t, step.Client.Get(context.TODO(), testutils.ObjectKey(podToDelete2), podToDelete2))

	assert.Nil(t, step.DeleteExisting(namespace))

	assert.Nil(t, step.Client.Get(context.TODO(), testutils.ObjectKey(podToKeep), podToKeep))
	assert.True(t, k8serrors.IsNotFound(step.Client.Get(context.TODO(), testutils.ObjectKey(podToDelete), podToDelete)))
	assert.True(t, k8serrors.IsNotFound(step.Client.Get(context.TODO(), testutils.ObjectKey(podToDelete2), podToDelete2)))
}
