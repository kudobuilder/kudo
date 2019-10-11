package task

import (
	"fmt"
	"testing"

	"github.com/kudobuilder/kudo/pkg/util/template"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestApplyTask_Run(t *testing.T) {
	tests := []struct {
		name    string
		task    ApplyTask
		want    bool
		wantErr bool
	}{
		{
			name: "succeeds when there is nothing to apply",
			task: ApplyTask{
				Name:      "install",
				Resources: []string{},
			},
			want:    true,
			wantErr: false,
		},
	}

	ctx := Context{
		Client:     fake.NewFakeClientWithScheme(scheme.Scheme),
		Enhancer:   &testKubernetesObjectEnhancer{},
		Meta:       ExecutionMetadata{},
		Templates:  nil,
		Parameters: nil,
	}
	for _, tt := range tests {
		got, err := tt.task.Run(ctx)
		assert.True(t, tt.want == got, fmt.Sprintf("%s failed: want = %t, wantErr = %v", tt.name, got, err))
		if tt.wantErr {
			assert.Error(t, err)
		}
	}
}

type testKubernetesObjectEnhancer struct{}

func (k *testKubernetesObjectEnhancer) ApplyConventionsToTemplates(templates map[string]string, metadata ExecutionMetadata) ([]runtime.Object, error) {
	result := make([]runtime.Object, len(templates))
	for _, t := range templates {
		objsToAdd, err := template.ParseKubernetesObjects(t)
		if err != nil {
			return nil, errors.Wrapf(err, "error parsing kubernetes objects after applying kustomize")
		}
		result = append(result, objsToAdd[0])
	}
	return result, nil
}
