package webhook

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/util/convert"
)

func TestValidateUpdate(t *testing.T) {
	// because Go doesn't let us take a pointer of a literal
	deploy := "deploy"
	update := "update"

	ov := &v1beta1.OperatorVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-operator", Namespace: "default"},
		TypeMeta:   metav1.TypeMeta{Kind: "OperatorVersion", APIVersion: "kudo.dev/v1beta1"},
		Spec: v1beta1.OperatorVersionSpec{
			Plans: map[string]v1beta1.Plan{"deploy": {}, "update": {}},
			Parameters: []v1beta1.Parameter{
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

	idle := &v1beta1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: v1beta1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "foo-operator",
			},
			Parameters: map[string]string{
				"foo": "foo",
			},
		},
		Status: v1beta1.InstanceStatus{},
	}

	scheduled := idle.DeepCopy()
	scheduled.Spec.PlanExecution = v1beta1.PlanExecution{PlanName: deploy}

	upgraded := idle.DeepCopy()
	upgraded.Spec.OperatorVersion = v1.ObjectReference{Name: "foo-operator-2.0"}

	tests := []struct {
		name    string
		new     *v1beta1.Instance
		old     *v1beta1.Instance
		ov      *v1beta1.OperatorVersion
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
			new: func() *v1beta1.Instance {
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
			new: func() *v1beta1.Instance {
				i := scheduled.DeepCopy()
				i.Spec.PlanExecution = v1beta1.PlanExecution{PlanName: "update"}
				return i
			}(),
			ov:      ov,
			wantErr: true,
		},
		{
			name: "canceling an existing plan directly is NOT allowed",
			old:  scheduled,
			new: func() *v1beta1.Instance {
				i := scheduled.DeepCopy()
				i.Spec.PlanExecution = v1beta1.PlanExecution{PlanName: ""}
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
			new: func() *v1beta1.Instance {
				i := upgraded.DeepCopy()
				i.Spec.PlanExecution = v1beta1.PlanExecution{PlanName: deploy}
				return i
			}(),
			ov:      ov,
			want:    nil,
			wantErr: true,
		},
		{
			name: "parameter update on an idle instance IS allowed",
			old:  idle,
			new: func() *v1beta1.Instance {
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
			new: func() *v1beta1.Instance {
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
			new: func() *v1beta1.Instance {
				i := scheduled.DeepCopy()
				i.Spec.Parameters = map[string]string{"foo": "newFoo"}
				return i
			}(),
			ov:   ov,
			want: &deploy,
		},
		{
			name: "updating parameter on a scheduled instance IS NOT allowed when a different plan is triggered",
			old:  scheduled,
			new: func() *v1beta1.Instance {
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
			new: func() *v1beta1.Instance {
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
			new: func() *v1beta1.Instance {
				i := idle.DeepCopy()
				i.Spec.Parameters["bazz"] = "newBazz"
				return i
			}(),
			ov:   ov,
			want: nil,
		},
		{
			name: "parameter update together with an upgrade IS normally NOT allowed",
			old:  idle,
			new: func() *v1beta1.Instance {
				i := upgraded.DeepCopy()
				i.Spec.Parameters = map[string]string{"foo": "newFoo"}
				return i
			}(),
			ov:      ov,
			wantErr: true,
		},
		{
			name: "parameter update together with an upgrade IS allowed if update removes a parameter that doesn't exist in the new OV",
			old:  idle,
			new: func() *v1beta1.Instance {
				i := upgraded.DeepCopy()
				delete(i.Spec.Parameters, "foo") // removing from instance parameters
				return i
			}(),
			ov: func() *v1beta1.OperatorVersion {
				o := ov.DeepCopy()
				o.Spec.Parameters = o.Spec.Parameters[1:len(o.Spec.Parameters)] // "foo" is the first parameter in the array
				return o
			}(),
			want: &update,
		},
		{
			name: "parameter update together with a directly triggered plan IS NOT allowed",
			old:  idle,
			new: func() *v1beta1.Instance {
				i := scheduled.DeepCopy()
				i.Spec.Parameters = map[string]string{"foo": "newFoo"}
				return i
			}(),
			ov:      ov,
			wantErr: true,
		},
		{
			name: "parameter update together with a directly triggered plan IS NOT allowed if different plans are triggered",
			old:  idle,
			new: func() *v1beta1.Instance {
				i := scheduled.DeepCopy()
				i.Spec.Parameters["bar"] = "newBar"
				return i
			}(),
			ov:      ov,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := admitUpdate(tt.old, tt.new, tt.ov)
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
	return "<nil>"
}

func Test_triggeredPlan(t *testing.T) {
	ov := &v1beta1.OperatorVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-operator", Namespace: "default"},
		TypeMeta:   metav1.TypeMeta{Kind: "OperatorVersion", APIVersion: "kudo.dev/v1beta1"},
		Spec: v1beta1.OperatorVersionSpec{
			Plans: map[string]v1beta1.Plan{"deploy": {}, "update": {}, "backup": {}},
			Parameters: []v1beta1.Parameter{
				{
					Name:    "param",
					Default: convert.String("default"),
				},
			},
		},
	}

	update := "update"
	backup := "backup"
	deploy := "deploy"

	tests := []struct {
		name    string
		params  []v1beta1.Parameter
		ov      *v1beta1.OperatorVersion
		want    *string
		wantErr bool
	}{
		{
			name:    "no change doesn't trigger anything",
			params:  []v1beta1.Parameter{},
			ov:      ov,
			want:    nil,
			wantErr: false,
		},
		{
			name:    "param without an explicit trigger, triggers update plan",
			params:  []v1beta1.Parameter{{Name: "foo"}},
			ov:      ov,
			want:    &update,
			wantErr: false,
		},
		{
			name:    "param with an explicit trigger",
			params:  []v1beta1.Parameter{{Name: "foo", Trigger: "backup"}},
			ov:      ov,
			want:    &backup,
			wantErr: false,
		},
		{
			name:    "two params with the same triggers",
			params:  []v1beta1.Parameter{{Name: "foo", Trigger: "deploy"}, {Name: "bar", Trigger: "deploy"}},
			ov:      ov,
			want:    &deploy,
			wantErr: false,
		},
		{
			name:    "two params with conflicting triggers lead to an error",
			params:  []v1beta1.Parameter{{Name: "foo", Trigger: "deploy"}, {Name: "bar", Trigger: "update"}},
			ov:      ov,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "params triggering a non-existing plan",
			params:  []v1beta1.Parameter{{Name: "foo", Trigger: "fake"}},
			ov:      ov,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := triggeredPlan(tt.params, tt.ov)
			if (err != nil) != tt.wantErr {
				t.Errorf("triggeredPlan() error = %v, wantErr %v, got = %s", err, tt.wantErr, stringPtrToString(got))
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
