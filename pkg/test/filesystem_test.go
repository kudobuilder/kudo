package test

import (
	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

// Verify the test state as loaded from disk.
// Each test provides a path to a set of test cases and their rendered result.
func TestLoadTestCases(t *testing.T) {
	for _, tt := range []struct {
		path      string
		testCases []TestCase
	}{
		{
			"./tests/kafka-upgrade/",
			[]TestCase{
				{
					Name:  "kafka-install",
					Index: 0,
					Apply: []runtime.Object{
						&unstructured.Unstructured{
							Object: map[string]interface{}{
								"apiVersion": "v1",
								"kind":       "Pod",
								"metadata": map[string]interface{}{
									"name": "test",
								},
								"spec": map[string]interface{}{
									"restartPolicy": "Never",
									"containers": []interface{}{
										map[string]interface{}{
											"command": []interface{}{
												"/usr/bin/tail", "-f", "/dev/null",
											},
											"image": "alpine",
											"name":  "test",
										},
									},
								},
							},
						},
						&unstructured.Unstructured{
							Object: map[string]interface{}{
								"apiVersion": "v1",
								"kind":       "Pod",
								"metadata": map[string]interface{}{
									"name": "test2",
								},
								"spec": map[string]interface{}{
									"restartPolicy": "Never",
									"containers": []interface{}{
										map[string]interface{}{
											"command": []interface{}{
												"/bin/cat", "/etc/non/existant",
											},
											"image": "alpine",
											"name":  "test",
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
									"name": "test",
								},
								"status": map[string]interface{}{
									"phase": "Running",
								},
							},
						},
						&unstructured.Unstructured{
							Object: map[string]interface{}{
								"apiVersion": "v1",
								"kind":       "Pod",
								"metadata": map[string]interface{}{
									"name": "test2",
								},
								"status": map[string]interface{}{
									"phase": "Failed",
								},
							},
						},
					},
					Errors: []runtime.Object{},
				},
				{
					Name:  "kafka-upgrade",
					Index: 1,
					Apply: []runtime.Object{
						&unstructured.Unstructured{
							Object: map[string]interface{}{
								"apiVersion": "kudo.k8s.io/v1alpha1",
								"kind":       "Instance",
								"metadata": map[string]interface{}{
									"name": "kafka",
								},
								"spec": map[string]interface{}{
									"frameworkVersion": map[string]interface{}{
										"name": "kafka-2.12-2.4.0",
									},
								},
							},
						},
					},
					Asserts: []runtime.Object{
						&unstructured.Unstructured{
							Object: map[string]interface{}{
								"apiVersion": "kudo.k8s.io/v1alpha1",
								"kind":       "Instance",
								"metadata": map[string]interface{}{
									"name": "kafka",
								},
								"status": map[string]interface{}{
									"running": "success",
									"version": "2.12-2.4.0",
								},
							},
						},
					},
					Errors: []runtime.Object{},
				},
			},
		},
		{
			"./tests/with-overrides/",
			[]TestCase{
				{
					Name:  "with-test-case-name-override",
					Index: 0,
					Case: &kudo.TestCase{
						ObjectMeta: metav1.ObjectMeta{
							Name: "with-test-case-name-override",
						},
						TypeMeta: metav1.TypeMeta{
							Kind:       "TestCase",
							APIVersion: "kudo.k8s.io/v1alpha1",
						},
						Index: 0,
					},
					Apply: []runtime.Object{
						&unstructured.Unstructured{
							Object: map[string]interface{}{
								"apiVersion": "v1",
								"kind":       "Pod",
								"metadata": map[string]interface{}{
									"name": "test",
								},
								"spec": map[string]interface{}{
									"restartPolicy": "Never",
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
									"name": "test",
								},
								"status": map[string]interface{}{
									"phase": "Running",
								},
							},
						},
					},
					Errors: []runtime.Object{},
				},
				{
					Name:  "test-assert",
					Index: 1,
					Case: &kudo.TestCase{
						TypeMeta: metav1.TypeMeta{
							Kind:       "TestCase",
							APIVersion: "kudo.k8s.io/v1alpha1",
						},
						Index: 1,
						Delete: []corev1.ObjectReference{
							{
								Kind: "Pod",
								Name: "test",
							},
						},
					},
					Assert: &kudo.TestAssert{
						TypeMeta: metav1.TypeMeta{
							Kind:       "TestAssert",
							APIVersion: "kudo.k8s.io/v1alpha1",
						},
						Timeout: 20,
					},
					Apply: []runtime.Object{
						&unstructured.Unstructured{
							Object: map[string]interface{}{
								"apiVersion": "v1",
								"kind":       "Pod",
								"metadata": map[string]interface{}{
									"name": "test2",
								},
								"spec": map[string]interface{}{
									"restartPolicy": "Never",
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
									"name": "test2",
								},
								"status": map[string]interface{}{
									"phase": "Running",
								},
							},
						},
					},
					Errors: []runtime.Object{},
				},
				{
					Name:  "pod",
					Index: 2,
					Apply: []runtime.Object{
						&unstructured.Unstructured{
							Object: map[string]interface{}{
								"apiVersion": "v1",
								"kind":       "Pod",
								"metadata": map[string]interface{}{
									"name": "test",
								},
								"status": map[string]interface{}{
									"phase": "Running",
								},
							},
						},
						&unstructured.Unstructured{
							Object: map[string]interface{}{
								"apiVersion": "v1",
								"kind":       "Pod",
								"metadata": map[string]interface{}{
									"name": "test2",
								},
								"status": map[string]interface{}{
									"phase": "Running",
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
									"name": "test",
								},
								"status": map[string]interface{}{
									"phase": "Running",
								},
							},
						},
					},
					Errors: []runtime.Object{},
				},
				{
					Name:  "name-overriden",
					Index: 3,
					Case: &kudo.TestCase{
						ObjectMeta: metav1.ObjectMeta{
							Name: "name-overriden",
						},
						TypeMeta: metav1.TypeMeta{
							Kind:       "TestCase",
							APIVersion: "kudo.k8s.io/v1alpha1",
						},
						Index: 3,
					},
					Apply: []runtime.Object{
						&unstructured.Unstructured{
							Object: map[string]interface{}{
								"apiVersion": "v1",
								"kind":       "Pod",
								"metadata": map[string]interface{}{
									"name": "test",
								},
								"spec": map[string]interface{}{
									"restartPolicy": "Never",
								},
							},
						},
						&unstructured.Unstructured{
							Object: map[string]interface{}{
								"apiVersion": "v1",
								"kind":       "Pod",
								"metadata": map[string]interface{}{
									"name": "test2",
								},
								"spec": map[string]interface{}{
									"restartPolicy": "Never",
								},
							},
						},
					},
					Asserts: []runtime.Object{},
					Errors:  []runtime.Object{},
				},
			},
		},
	} {
		t.Run(tt.path, func(t *testing.T) {
			test := &Test{Dir: tt.path}

			err := test.LoadTestCases()
			assert.Nil(t, err)

			testCasesVal := []TestCase{}
			for _, testCase := range test.Cases {
				testCasesVal = append(testCasesVal, *testCase)
			}

			assert.Equal(t, len(tt.testCases), len(testCasesVal))
			for index := range tt.testCases {
				assert.Equal(t, tt.testCases[index], testCasesVal[index])
			}
		})
	}
}

func TestCollectTestCaseFiles(t *testing.T) {
	for _, tt := range []struct {
		path     string
		expected map[int64][]string
	}{
		{
			"tests/kafka-upgrade",
			map[int64][]string{
				int64(0): {
					"tests/kafka-upgrade/00-assert.yaml",
					"tests/kafka-upgrade/00-kafka-install.yaml",
				},
				int64(1): {
					"tests/kafka-upgrade/01-assert.yaml",
					"tests/kafka-upgrade/01-kafka-upgrade.yaml",
				},
			},
		},
		{
			"tests/with-overrides",
			map[int64][]string{
				int64(0): {
					"tests/with-overrides/00-assert.yaml",
					"tests/with-overrides/00-test-case.yaml",
				},
				int64(1): {
					"tests/with-overrides/01-assert.yaml",
					"tests/with-overrides/01-test-assert.yaml",
				},
				int64(2): {
					"tests/with-overrides/02-directory/assert.yaml",
					"tests/with-overrides/02-directory/pod.yaml",
					"tests/with-overrides/02-directory/pod2.yaml",
				},
				int64(3): {
					"tests/with-overrides/03-assert.yaml",
					"tests/with-overrides/03-pod.yaml",
					"tests/with-overrides/03-pod2.yaml",
				},
			},
		},
	} {
		t.Run(tt.path, func(t *testing.T) {
			test := &Test{Dir: tt.path}
			testCaseFiles, err := test.CollectTestCaseFiles()
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, testCaseFiles)
		})
	}
}
