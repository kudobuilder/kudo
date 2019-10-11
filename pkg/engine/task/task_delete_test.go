package task

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDeleteTask_Run(t *testing.T) {
	tests := []struct {
		name    string
		task    DeleteTask
		want    bool
		wantErr bool
	}{
		{
			name: "succeeds when there is nothing to delete",
			task: DeleteTask{
				Name:      "uninstall",
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
