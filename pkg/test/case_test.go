package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	harness "github.com/kudobuilder/kudo/pkg/apis/testharness/v1beta1"
	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
)

// Verify the test state as loaded from disk.
// Each test provides a path to a set of test steps and their rendered result.
func TestLoadTestSteps(t *testing.T) {
	for _, tt := range []struct {
		path      string
		testSteps []Step
	}{
		{
			"test_data/with-overrides/",
			[]Step{
				{
					Name:  "with-test-step-name-override",
					Index: 0,
					Step: &harness.TestStep{
						ObjectMeta: metav1.ObjectMeta{
							Name: "with-test-step-name-override",
						},
						TypeMeta: metav1.TypeMeta{
							Kind:       "TestStep",
							APIVersion: "kudo.dev/v1beta1",
						},
						Index: 0,
					},
					Apply: []runtime.Object{
						testutils.WithSpec(t, testutils.NewPod("test", ""), map[string]interface{}{
							"restartPolicy": "Never",
							"containers": []map[string]interface{}{
								{
									"name":  "nginx",
									"image": "nginx:1.7.9",
								},
							},
						}),
					},
					Asserts: []runtime.Object{
						testutils.WithStatus(t, testutils.NewPod("test", ""), map[string]interface{}{
							"qosClass": "BestEffort",
						}),
					},
					Errors: []runtime.Object{},
				},
				{
					Name:  "test-assert",
					Index: 1,
					Step: &harness.TestStep{
						TypeMeta: metav1.TypeMeta{
							Kind:       "TestStep",
							APIVersion: "kudo.dev/v1beta1",
						},
						Index: 1,
						Delete: []harness.ObjectReference{
							{
								ObjectReference: corev1.ObjectReference{
									APIVersion: "v1",
									Kind:       "Pod",
									Name:       "test",
								},
							},
						},
					},
					Assert: &harness.TestAssert{
						TypeMeta: metav1.TypeMeta{
							Kind:       "TestAssert",
							APIVersion: "kudo.dev/v1beta1",
						},
						Timeout: 20,
					},
					Apply: []runtime.Object{
						testutils.WithSpec(t, testutils.NewPod("test2", ""), map[string]interface{}{
							"restartPolicy": "Never",
							"containers": []map[string]interface{}{
								{
									"name":  "nginx",
									"image": "nginx:1.7.9",
								},
							},
						}),
					},
					Asserts: []runtime.Object{
						testutils.WithStatus(t, testutils.NewPod("test2", ""), map[string]interface{}{
							"qosClass": "BestEffort",
						}),
					},
					Errors: []runtime.Object{},
				},
				{
					Name:  "pod",
					Index: 2,
					Apply: []runtime.Object{
						testutils.WithSpec(t, testutils.NewPod("test4", ""), map[string]interface{}{
							"containers": []map[string]interface{}{
								{
									"name":  "nginx",
									"image": "nginx:1.7.9",
								},
							},
						}),
						testutils.WithSpec(t, testutils.NewPod("test3", ""), map[string]interface{}{
							"containers": []map[string]interface{}{
								{
									"name":  "nginx",
									"image": "nginx:1.7.9",
								},
							},
						}),
					},
					Asserts: []runtime.Object{
						testutils.WithStatus(t, testutils.NewPod("test3", ""), map[string]interface{}{
							"qosClass": "BestEffort",
						}),
					},
					Errors: []runtime.Object{},
				},
				{
					Name:  "name-overridden",
					Index: 3,
					Step: &harness.TestStep{
						ObjectMeta: metav1.ObjectMeta{
							Name: "name-overridden",
						},
						TypeMeta: metav1.TypeMeta{
							Kind:       "TestStep",
							APIVersion: "kudo.dev/v1beta1",
						},
						Index: 3,
					},
					Apply: []runtime.Object{
						testutils.WithSpec(t, testutils.NewPod("test6", ""), map[string]interface{}{
							"restartPolicy": "Never",
							"containers": []map[string]interface{}{
								{
									"name":  "nginx",
									"image": "nginx:1.7.9",
								},
							},
						}),
						testutils.WithSpec(t, testutils.NewPod("test5", ""), map[string]interface{}{
							"restartPolicy": "Never",
							"containers": []map[string]interface{}{
								{
									"name":  "nginx",
									"image": "nginx:1.7.9",
								},
							},
						}),
					},
					Asserts: []runtime.Object{
						testutils.WithSpec(t, testutils.NewPod("test5", ""), map[string]interface{}{
							"restartPolicy": "Never",
						}),
					},
					Errors: []runtime.Object{},
				},
			},
		},
		{
			"test_data/list-pods",
			[]Step{
				{
					Name:  "pod",
					Index: 0,
					Apply: []runtime.Object{
						&unstructured.Unstructured{
							Object: map[string]interface{}{
								"apiVersion": "v1",
								"kind":       "Pod",
								"metadata": map[string]interface{}{
									"name": "pod-1",
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
					Asserts: []runtime.Object{
						&unstructured.Unstructured{
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
					Errors: []runtime.Object{},
				},
			},
		},
	} {
		t.Run(tt.path, func(t *testing.T) {
			test := &Case{Dir: tt.path, Logger: testutils.NewTestLogger(t, tt.path)}

			err := test.LoadTestSteps()
			assert.Nil(t, err)

			testStepsVal := []Step{}
			for _, testStep := range test.Steps {
				testStepsVal = append(testStepsVal, *testStep)
			}

			assert.Equal(t, len(tt.testSteps), len(testStepsVal))
			for index := range tt.testSteps {
				tt.testSteps[index].Dir = tt.path
				assert.Equal(t, tt.testSteps[index].Apply, testStepsVal[index].Apply)
				assert.Equal(t, tt.testSteps[index].Asserts, testStepsVal[index].Asserts)
				assert.Equal(t, tt.testSteps[index].Errors, testStepsVal[index].Errors)
				assert.Equal(t, tt.testSteps[index].Step, testStepsVal[index].Step)
				assert.Equal(t, tt.testSteps[index].Dir, testStepsVal[index].Dir)
				assert.Equal(t, tt.testSteps[index], testStepsVal[index])
			}
		})
	}
}

func TestCollectTestStepFiles(t *testing.T) {
	for _, tt := range []struct {
		path     string
		expected map[int64][]string
	}{
		{
			"test_data/with-overrides",
			map[int64][]string{
				int64(0): {
					"test_data/with-overrides/00-assert.yaml",
					"test_data/with-overrides/00-test-step.yaml",
				},
				int64(1): {
					"test_data/with-overrides/01-assert.yaml",
					"test_data/with-overrides/01-test-assert.yaml",
				},
				int64(2): {
					"test_data/with-overrides/02-directory/assert.yaml",
					"test_data/with-overrides/02-directory/pod.yaml",
					"test_data/with-overrides/02-directory/pod2.yaml",
				},
				int64(3): {
					"test_data/with-overrides/03-assert.yaml",
					"test_data/with-overrides/03-pod.yaml",
					"test_data/with-overrides/03-pod2.yaml",
				},
			},
		},
		{
			"test_data/list-pods",
			map[int64][]string{
				int64(0): {
					"test_data/list-pods/00-assert.yaml",
					"test_data/list-pods/00-pod.yaml",
				},
			},
		},
	} {
		t.Run(tt.path, func(t *testing.T) {
			test := &Case{Dir: tt.path, Logger: testutils.NewTestLogger(t, tt.path)}
			testStepFiles, err := test.CollectTestStepFiles()
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, testStepFiles)
		})
	}
}
