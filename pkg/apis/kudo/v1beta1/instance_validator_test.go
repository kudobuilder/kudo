package v1beta1

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateUpdate(t *testing.T) {
	runningInstance := Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "test-1.0",
			},
		},
		Status: InstanceStatus{
			AggregatedStatus: AggregatedStatus{
				Status:         ExecutionInProgress,
				ActivePlanName: "deploy",
			},
		},
	}

	tests := []struct {
		name          string
		new           Instance
		old           Instance
		expectedError error
	}{
		{"no change", runningInstance, runningInstance, nil},
		{"change in labels is allowed on running instance", runningInstance, func() Instance {
			updatedInstance := runningInstance.DeepCopy()
			updatedInstance.ObjectMeta.Labels = map[string]string{"label": "label2"}
			return *updatedInstance
		}(), nil},
		{"change in spec is not allowed on running instance", runningInstance, func() Instance {
			updatedInstance := runningInstance.DeepCopy()
			updatedInstance.Spec.Parameters = map[string]string{"newparam": "newvalue"}
			return *updatedInstance
		}(), errors.New("cannot update Instance test/test right now, there's plan deploy in progress")},
	}

	for _, tt := range tests {
		err := validateUpdate(&tt.old, &tt.new)
		assert.Equal(t, tt.expectedError, err)
	}
}
