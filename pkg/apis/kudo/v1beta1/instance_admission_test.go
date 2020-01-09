package v1beta1

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

func TestValidateUpdate(t *testing.T) {
	// because Go doesn't let us take a pointer of a literal
	deploy := "deploy"
	update := "update"

	ov := &OperatorVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-operator", Namespace: "default"},
		TypeMeta:   metav1.TypeMeta{Kind: "OperatorVersion", APIVersion: "kudo.dev/v1beta1"},
		Spec: OperatorVersionSpec{
			Plans: map[string]Plan{"deploy": {}, "update": {}},
			Parameters: []Parameter{
				{
					Name:    "foo",
					Trigger: deploy,
				},
				{
					Name:    "other-foo",
					Trigger: deploy,
				},
				{
					Name:    "bar",
					Trigger: update,
				},
			},
		},
	}

	idle := &Instance{
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
			Parameters: map[string]string{
				"foo": "foo",
			},
		},
		Status: InstanceStatus{},
	}

	scheduled := idle.DeepCopy()
	scheduled.Spec.PlanExecution = PlanExecution{PlanName: deploy}

	upgraded := idle.DeepCopy()
	upgraded.Spec.OperatorVersion = v1.ObjectReference{Name: "foo-operator-2.0"}

	tests := []struct {
		name    string
		new     *Instance
		old     *Instance
		ov      *OperatorVersion
		want    *string
		wantErr bool
	}{
		{
			name: "no change is a noop",
			old:  idle,
			new:  idle,
			ov:   ov,
		},
		{
			name: "change in labels does not trigger a plan",
			old:  scheduled,
			new: func() *Instance {
				i := scheduled.DeepCopy()
				i.ObjectMeta.Labels = map[string]string{"label": "label2"}
				return i
			}(),
			ov:   ov,
			want: nil,
		},
		{
			name: "triggering a plan directly IS allowed when NO plan is scheduled",
			old:  idle,
			new:  scheduled,
			ov:   ov,
			want: &deploy,
		},
		{
			name: "triggering the same plan directly IS allowed",
			old:  scheduled,
			new:  scheduled,
			ov:   ov,
			want: nil,
		},
		{
			name: "overriding an existing plan directly is NOT allowed",
			old:  scheduled,
			new: func() *Instance {
				i := scheduled.DeepCopy()
				i.Spec.PlanExecution = PlanExecution{PlanName: "update"}
				return i
			}(),
			ov:      ov,
			wantErr: true,
		},
		{
			name: "canceling an existing plan directly is NOT allowed",
			old:  scheduled,
			new: func() *Instance {
				i := scheduled.DeepCopy()
				i.Spec.PlanExecution = PlanExecution{PlanName: ""}
				return i
			}(),
			ov:      ov,
			wantErr: true,
		},
		{
			name: "upgrade triggered on an idle instance IS allowed",
			old:  idle,
			new:  upgraded,
			ov:   ov,
			want: &update, // 'update' is a fallback plan when 'upgrade' does not exist
		},
		{
			name:    "upgrade triggered on a scheduled instance IS NOT allowed",
			old:     scheduled,
			new:     upgraded,
			ov:      ov,
			want:    nil,
			wantErr: true,
		},
		{
			name: "upgrade triggered on an idle instance together with another plan IS NOT allowed",
			old:  idle,
			new: func() *Instance {
				i := upgraded.DeepCopy()
				i.Spec.PlanExecution = PlanExecution{PlanName: deploy}
				return i
			}(),
			ov:      ov,
			want:    nil,
			wantErr: true,
		},
		{
			name: "parameter update on an idle instance IS allowed",
			old:  idle,
			new: func() *Instance {
				i := idle.DeepCopy()
				i.Spec.Parameters = map[string]string{"foo": "newFoo"}
				return i
			}(),
			ov:   ov,
			want: &deploy,
		},
		{
			name: "updating multiple parameters on an idle instance IS allowed when the same plan is triggered",
			old:  idle,
			new: func() *Instance {
				i := idle.DeepCopy()
				i.Spec.Parameters = map[string]string{"foo": "newFoo", "other-foo": "newOtherFoo"}
				return i
			}(),
			ov:   ov,
			want: &deploy,
		},
		{
			name: "parameter update on a scheduled instance IS allowed when the same plan is triggered",
			old:  scheduled,
			new: func() *Instance {
				i := scheduled.DeepCopy()
				i.Spec.Parameters = map[string]string{"foo": "newFoo"}
				return i
			}(),
			ov:   ov,
			want: nil,
		},
		{
			name: "updating parameter on a scheduled instance IS NOT allowed when a different plan is triggered",
			old:  scheduled,
			new: func() *Instance {
				i := scheduled.DeepCopy()
				i.Spec.Parameters = map[string]string{"bar": "newBar"}
				return i
			}(),
			ov:      ov,
			wantErr: true,
		},
		{
			name: "parameter update triggering multiple distinct plans IS NOT allowed",
			old:  idle,
			new: func() *Instance {
				i := idle.DeepCopy()
				i.Spec.Parameters = map[string]string{"foo": "newFoo", "bar": "newBar"}
				return i
			}(),
			ov:      ov,
			wantErr: true,
		},
		{
			name: "parameter update triggering a non-existing ov plan IS allowed but will NOT trigger a plan",
			old:  idle,
			new: func() *Instance {
				i := idle.DeepCopy()
				i.Spec.Parameters["bazz"] = "newBazz"
				return i
			}(),
			ov:   ov,
			want: nil,
		},
		{
			name: "parameter update together with an upgrade IS allowed",
			old:  idle,
			new: func() *Instance {
				i := upgraded.DeepCopy()
				i.Spec.Parameters = map[string]string{"foo": "newFoo", "bar": "newBar"}
				return i
			}(),
			ov:   ov,
			want: &update,
		},
		{
			name: "parameter update together with a directly triggered plan IS allowed if the same plan is triggered",
			old:  idle,
			new: func() *Instance {
				i := scheduled.DeepCopy()
				i.Spec.Parameters = map[string]string{"foo": "newFoo"}
				return i
			}(),
			ov:   ov,
			want: &deploy,
		},
		{
			name: "parameter update together with a directly triggered plan IS NOT allowed if different plans are triggered",
			old:  idle,
			new: func() *Instance {
				i := scheduled.DeepCopy()
				i.Spec.Parameters["bar"] = "newBar"
				return i
			}(),
			ov:      ov,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateUpdate(tt.old, tt.new, tt.ov)
			assert.Equal(t, tt.wantErr, err != nil, "expected an error: %v but got: %v", tt.wantErr, err)
			if err != nil {
				log.Printf("err: %v", err)
			}
			assert.Equal(t, tt.want, got, "expected '%s' plan triggered but got: '%s'", stringPtrToString(tt.want), stringPtrToString(got))
		})
	}
}

func stringPtrToString(p *string) string {
	if p != nil {
		return *p
	}
	return "(nil)"
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
			name:    "no change doesn't trigger anything",
			params:  []Parameter{},
			ov:      ov,
			want:    "",
			wantErr: false,
		},
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
