package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kudobuilder/kudo/pkg/util/kudo"
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
		want          string
		expectedError error
	}{
		{
			name: "no change",
			new:  runningInstance,
			old:  runningInstance,
			ov:   ov,
			want: "",
		},
		{
			name: "change in labels is allowed on running instance",
			new:  runningInstance,
			old: func() Instance {
				updatedInstance := runningInstance.DeepCopy()
				updatedInstance.ObjectMeta.Labels = map[string]string{"label": "label2"}
				return *updatedInstance
			}(),
			ov:   ov,
			want: "",
		},
		//{name: "change in spec is not allowed on running instance", new: runningInstance, old: func() Instance {
		//	updatedInstance := runningInstance.DeepCopy()
		//	updatedInstance.Spec.Parameters = map[string]string{"newparam": "newvalue"}
		//	return *updatedInstance
		//}(), ov: ov, expectedError: errors.New("cannot update Instance test/test right now, there's plan deploy in progress")},
	}

	for _, tt := range tests {
		got, err := validateUpdate(&tt.old, &tt.new, tt.ov)
		assert.Equal(t, tt.expectedError, err)
		assert.Equal(t, tt.want, got)
	}
}

func Test_parameterDiffPlan(t *testing.T) {
	ov := &OperatorVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-operator", Namespace: "default"},
		TypeMeta:   metav1.TypeMeta{Kind: "OperatorVersion", APIVersion: "kudo.dev/v1beta1"},
		Spec: OperatorVersionSpec{
			Plans: map[string]Plan{"deploy": {}, "update": {}, "backup": {}},
			Parameters: []Parameter{
				{
					Name:    "param",
					Default: kudo.String("default"),
				},
			},
		},
	}

	tests := []struct {
		name    string
		params  []Parameter
		ov      *OperatorVersion
		want    string
		wantErr bool
	}{
		{
			name:    "param without an explicit trigger, triggers update plan",
			params:  []Parameter{{Name: "foo"}},
			ov:      ov,
			want:    "update",
			wantErr: false,
		},
		{
			name:    "param with an explicit trigger",
			params:  []Parameter{{Name: "foo", Trigger: "backup"}},
			ov:      ov,
			want:    "backup",
			wantErr: false,
		},
		{
			name:    "two params with the same triggers",
			params:  []Parameter{{Name: "foo", Trigger: "deploy"}, {Name: "bar", Trigger: "deploy"}},
			ov:      ov,
			want:    "deploy",
			wantErr: false,
		},
		{
			name:    "two params with conflicting triggers lead to an error",
			params:  []Parameter{{Name: "foo", Trigger: "deploy"}, {Name: "bar", Trigger: "update"}},
			ov:      ov,
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parameterDiffPlan(tt.params, tt.ov)
			if (err != nil) != tt.wantErr {
				t.Errorf("parameterDiffPlan() error = %v, wantErr %v, got = %s", err, tt.wantErr, got)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
