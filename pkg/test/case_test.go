package test

import (
	"testing"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
					Step: &kudo.TestStep{
						ObjectMeta: metav1.ObjectMeta{
							Name: "with-test-step-name-override",
						},
						TypeMeta: metav1.TypeMeta{
							Kind:       "TestStep",
							APIVersion: "kudo.k8s.io/v1alpha1",
						},
						Index: 0,
					},
					Apply: []runtime.Object{
						testutils.WithSpec(testutils.NewPod("test", ""), map[string]interface{}{
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
						testutils.WithStatus(testutils.NewPod("test", ""), map[string]interface{}{
							"qosClass": "BestEffort",
						}),
					},
					Errors: []runtime.Object{},
				},
				{
					Name:  "test-assert",
					Index: 1,
					Step: &kudo.TestStep{
						TypeMeta: metav1.TypeMeta{
							Kind:       "TestStep",
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
						testutils.WithSpec(testutils.NewPod("test2", ""), map[string]interface{}{
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
						testutils.WithStatus(testutils.NewPod("test2", ""), map[string]interface{}{
							"qosClass": "BestEffort",
						}),
					},
					Errors: []runtime.Object{},
				},
				{
					Name:  "pod",
					Index: 2,
					Apply: []runtime.Object{
						testutils.WithSpec(testutils.NewPod("test4", ""), map[string]interface{}{
							"containers": []map[string]interface{}{
								{
									"name":  "nginx",
									"image": "nginx:1.7.9",
								},
							},
						}),
						testutils.WithSpec(testutils.NewPod("test3", ""), map[string]interface{}{
							"containers": []map[string]interface{}{
								{
									"name":  "nginx",
									"image": "nginx:1.7.9",
								},
							},
						}),
					},
					Asserts: []runtime.Object{
						testutils.WithStatus(testutils.NewPod("test3", ""), map[string]interface{}{
							"qosClass": "BestEffort",
						}),
					},
					Errors: []runtime.Object{},
				},
				{
					Name:  "name-overriden",
					Index: 3,
					Step: &kudo.TestStep{
						ObjectMeta: metav1.ObjectMeta{
							Name: "name-overriden",
						},
						TypeMeta: metav1.TypeMeta{
							Kind:       "TestStep",
							APIVersion: "kudo.k8s.io/v1alpha1",
						},
						Index: 3,
					},
					Apply: []runtime.Object{
						testutils.WithSpec(testutils.NewPod("test6", ""), map[string]interface{}{
							"restartPolicy": "Never",
							"containers": []map[string]interface{}{
								{
									"name":  "nginx",
									"image": "nginx:1.7.9",
								},
							},
						}),
						testutils.WithSpec(testutils.NewPod("test5", ""), map[string]interface{}{
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
						testutils.WithSpec(testutils.NewPod("test5", ""), map[string]interface{}{
							"restartPolicy": "Never",
						}),
					},
					Errors: []runtime.Object{},
				},
			},
		},
	} {
		t.Run(tt.path, func(t *testing.T) {
			test := &Case{Dir: tt.path}

			err := test.LoadTestSteps()
			assert.Nil(t, err)

			testStepsVal := []Step{}
			for _, testStep := range test.Steps {
				testStepsVal = append(testStepsVal, *testStep)
			}

			assert.Equal(t, len(tt.testSteps), len(testStepsVal))
			for index := range tt.testSteps {
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
	} {
		t.Run(tt.path, func(t *testing.T) {
			test := &Case{Dir: tt.path}
			testStepFiles, err := test.CollectTestStepFiles()
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, testStepFiles)
		})
	}
}
