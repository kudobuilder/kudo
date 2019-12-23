package v1beta1

import (
	"errors"
	"testing"

	"github.com/kudobuilder/kudo/pkg/util/kudo"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateUpdate(t *testing.T) {
	ov := &OperatorVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-operator", Namespace: "default"},
		TypeMeta:   metav1.TypeMeta{Kind: "OperatorVersion", APIVersion: "kudo.dev/v1beta1"},
		Spec: OperatorVersionSpec{
			Plans: map[string]Plan{"deploy": {}, "update": {}},
			Parameters: []Parameter{
				{
					Name:    "param",
					Default: kudo.String("default"),
				},
			},
		},
	}

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
				Name: "foo-operator",
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
		ov            *OperatorVersion
		expectedError error
	}{
		{name: "no change", new: runningInstance, old: runningInstance, ov: ov},
		{name: "change in labels is allowed on running instance", new: runningInstance, old: func() Instance {
			updatedInstance := runningInstance.DeepCopy()
			updatedInstance.ObjectMeta.Labels = map[string]string{"label": "label2"}
			return *updatedInstance
		}(), ov: ov},
		{name: "change in spec is not allowed on running instance", new: runningInstance, old: func() Instance {
			updatedInstance := runningInstance.DeepCopy()
			updatedInstance.Spec.Parameters = map[string]string{"newparam": "newvalue"}
			return *updatedInstance
		}(), ov: ov, expectedError: errors.New("cannot update Instance test/test right now, there's plan deploy in progress")},
	}

	for _, tt := range tests {
		_, err := validateUpdate(&tt.old, &tt.new, tt.ov)
		assert.Equal(t, tt.expectedError, err)
	}
}
