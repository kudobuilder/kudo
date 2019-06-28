package planexecution

import (
	"testing"

	testutils "github.com/kudobuilder/kudo/pkg/test/utils"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestPatchObject(t *testing.T) {
	tests := []struct {
		name     string
		actual   runtime.Object
		expected runtime.Object
		patch    string
	}{
		{
			name:     "empty patch",
			actual:   testutils.NewPod("hello", "world"),
			expected: testutils.NewPod("hello", "world"),
			patch:    `{"metadata": {"annotations": {}}, "spec": {}}`,
		},
		{
			name:     "add annotations",
			actual:   testutils.NewPod("hello", "world"),
			expected: testutils.WithAnnotations(testutils.NewPod("hello", "world"), map[string]string{"app": "nginx"}),
			patch:    `{"metadata": {"annotations": {"app":"nginx"}}, "spec": {}}`,
		},
		{
			name:     "add spec",
			actual:   testutils.NewPod("hello", "world"),
			expected: testutils.WithSpec(testutils.NewPod("hello", "world"), map[string]interface{}{"RestartPolicy": "Always"}),
			patch:    `{"metadata": {"annotations": {}}, "spec": {"RestartPolicy":"Always"}}`,
		},
		{
			name:     "remove annotations",
			actual:   testutils.WithAnnotations(testutils.NewPod("hello", "world"), map[string]string{"app": "nginx"}),
			expected: testutils.NewPod("hello", "world"),
			patch:    `{"metadata": {"annotations": {}}, "spec": {}}`,
		},
		{
			name:     "remove spec",
			actual:   testutils.WithSpec(testutils.NewPod("hello", "world"), map[string]interface{}{"RestartPolicy": "Always"}),
			expected: testutils.NewPod("hello", "world"),
			patch:    `{"metadata": {"annotations": {}}, "spec": {}}`,
		},
		{
			name:     "change annotations",
			actual:   testutils.WithAnnotations(testutils.NewPod("hello", "world"), map[string]string{"app": "memcached"}),
			expected: testutils.WithAnnotations(testutils.NewPod("hello", "world"), map[string]string{"app": "nginx"}),
			patch:    `{"metadata": {"annotations": {"app":"nginx"}}, "spec": {}}`,
		},
		{
			name:     "change spec",
			actual:   testutils.WithSpec(testutils.NewPod("hello", "world"), map[string]interface{}{"RestartPolicy": "Never"}),
			expected: testutils.WithSpec(testutils.NewPod("hello", "world"), map[string]interface{}{"RestartPolicy": "Always"}),
			patch:    `{"metadata": {"annotations": {}}, "spec": {"RestartPolicy":"Always"}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patch, err := patchObject(tt.expected, tt.actual)
			assert.Nil(t, err)
			assert.Equal(t, tt.patch, patch)
		})
	}
}
