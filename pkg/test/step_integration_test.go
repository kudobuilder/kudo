// +build integration

package test

import (
	"context"
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"

	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
	"github.com/stretchr/testify/assert"
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
			testName: "match object by labels",
			actual: []runtime.Object{
				testutils.WithSpec(testutils.WithLabels(testutils.NewPod("hello", ""), map[string]string{
					"app": "not-match",
				}), map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"image": "nginx:1.7.9",
							"name":  "nginx",
						},
					},
				}),
				testutils.WithSpec(testutils.WithLabels(testutils.NewPod("hello1", ""), map[string]string{
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
