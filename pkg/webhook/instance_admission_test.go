package webhook

import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/util/convert"
)

func TestValidateUpdate(t *testing.T) {
	deploy := v1beta1.DeployPlanName
	update := v1beta1.UpdatePlanName
	cleanup := v1beta1.CleanupPlanName
	backup := "backup"
	empty := ""

	testUUID := uuid.NewUUID()

	ov := &v1beta1.OperatorVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "foo-operator", Namespace: "default"},
		TypeMeta:   metav1.TypeMeta{Kind: "OperatorVersion", APIVersion: "kudo.dev/v1beta1"},
		Spec: v1beta1.OperatorVersionSpec{
			Plans: map[string]v1beta1.Plan{deploy: {}, update: {}, cleanup: {}, backup: {}},
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
				{
					Name:    "backup",
					Trigger: backup,
				},
				{
					Name:    "invalid",
					Trigger: "missing",
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
	scheduled.Spec.PlanExecution = v1beta1.PlanExecution{PlanName: deploy, UID: testUUID}

	upgraded := idle.DeepCopy()
	upgraded.Spec.OperatorVersion = v1.ObjectReference{Name: "foo-operator-2.0"}

	deleted := scheduled.DeepCopy()
	deleted.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Date(2019, 10, 17, 1, 1, 1, 1, time.UTC)}
	deleted.ObjectMeta.Finalizers = []string{"kudo.dev.instance.cleanup"}

	uninstalling := deleted.DeepCopy()
	uninstalling.Spec.PlanExecution.PlanName = cleanup

	tests := []struct {
		name    string
		new     *v1beta1.Instance
		old     *v1beta1.Instance
		ov      *v1beta1.OperatorVersion
		want    *string
		wantErr bool
	}{
		{
			name: "no change is a NOOP",
			old:  idle,
			new:  idle,
			ov:   ov,
		},
		{
			name: "change in labels does NOT trigger a plan",
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
			new: func() *v1beta1.Instance {
				i := scheduled.DeepCopy()
				i.Spec.PlanExecution = v1beta1.PlanExecution{PlanName: deploy, UID: "foo"} // a UID change will result in the same plan re-triggered
				return i
			}(),
			ov:   ov,
			want: &deploy,
		},
		{
			name: "overriding an existing plan directly is NOT allowed",
			old:  scheduled,
			new: func() *v1beta1.Instance {
				i := scheduled.DeepCopy()
				i.Spec.PlanExecution = v1beta1.PlanExecution{PlanName: update}
				return i
			}(),
			ov:      ov,
			wantErr: true,
		},
		{
			name: "cleanup plan CAN override an existing plan directly if the instance is being deleted",
			old:  deleted,
			new:  uninstalling,
			ov:   ov,
		},
		{
			name: "cleanup plan CAN NOT be overridden by any other plan if the instance is being deleted",
			old:  uninstalling,
			new: func() *v1beta1.Instance {
				i := uninstalling.DeepCopy()
				i.Spec.PlanExecution = v1beta1.PlanExecution{PlanName: deploy}
				return i
			}(),
			ov:      ov,
			wantErr: true,
		},
		{
			name: "cleanup plan CAN NOT be cancelled when the instance is being deleted",
			old:  uninstalling,
			new: func() *v1beta1.Instance {
				i := uninstalling.DeepCopy()
				i.Spec.PlanExecution = v1beta1.PlanExecution{PlanName: ""}
				return i
			}(),
			ov:      ov,
			wantErr: true,
		},
		{
			name: "plan execution IS reset when the plan is terminal",
			old:  scheduled,
			new: func() *v1beta1.Instance {
				i := scheduled.DeepCopy()
				i.Spec.PlanExecution.Status = v1beta1.ExecutionComplete
				return i
			}(),
			ov:   ov,
			want: &empty,
		},
		{
			name: "cleanup plan CAN NOT be triggered directly by the user",
			old:  idle,
			new: func() *v1beta1.Instance {
				i := idle.DeepCopy()
				i.Spec.PlanExecution = v1beta1.PlanExecution{PlanName: cleanup}
				return i
			}(),
			ov:      ov,
			wantErr: true,
		},
		{
			name: "updates are NOT allowed when the instance is being deleted",
			old:  deleted,
			new: func() *v1beta1.Instance {
				i := deleted.DeepCopy()
				i.Spec.Parameters = map[string]string{"foo": "newFoo"}
				return i
			}(),
			ov:      ov,
			wantErr: true,
		},
		{
			name: "upgrades are NOT allowed when the instance is being deleted",
			old:  deleted,
			new: func() *v1beta1.Instance {
				i := deleted.DeepCopy()
				i.Spec.OperatorVersion = v1.ObjectReference{Name: "foo-operator-2.0"}
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
			name: "parameter update triggering a non-existing OV parameter IS NOT allowed",
			old:  idle,
			new: func() *v1beta1.Instance {
				i := idle.DeepCopy()
				i.Spec.Parameters["bazz"] = "newBazz"
				return i
			}(),
			ov:      ov,
			wantErr: true,
		},
		{
			name: "parameter update triggering a non-existing OV plan IS NOT allowed",
			old:  idle,
			new: func() *v1beta1.Instance {
				i := idle.DeepCopy()
				i.Spec.Parameters["invalid"] = "invalid"
				return i
			}(),
			ov:      ov,
			wantErr: true,
		},
		{
			name: "parameter update together with an upgrade IS NOT allowed if a plan other than deploy is triggered",
			old:  idle,
			new: func() *v1beta1.Instance {
				i := upgraded.DeepCopy()
				i.Spec.Parameters = map[string]string{"backup": "back"}
				return i
			}(),
			ov:      ov,
			wantErr: true,
		},
		{
			name: "parameter update together with an upgrade IS allowed if deploy is triggered",
			old:  idle,
			new: func() *v1beta1.Instance {
				i := upgraded.DeepCopy()
				i.Spec.Parameters = map[string]string{"foo": "newFoo"}
				return i
			}(),
			ov:      ov,
			wantErr: false,
			want:    &update,
		},
		{
			name: "parameter update together with an upgrade IS allowed if update removes a parameter that doesn't exist in the new OV",
			old:  idle,
			new: func() *v1beta1.Instance {
				i := upgraded.DeepCopy()
				delete(i.Spec.Parameters, "backup") // removing from instance parameters
				return i
			}(),
			ov: func() *v1beta1.OperatorVersion {
				o := ov.DeepCopy()
				o.Spec.Parameters = o.Spec.Parameters[:len(o.Spec.Parameters)-1] // "backup" is the last parameter in the array
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
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := admitUpdate(tt.old, tt.new, tt.ov, nil)
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
					Default: convert.StringPtr("default"),
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
			got, err := triggeredByParameterUpdate(tt.params, tt.ov)
			if (err != nil) != tt.wantErr {
				t.Errorf("triggeredByParameterUpdate() error = %v, wantErr %v, got = %s", err, tt.wantErr, stringPtrToString(got))
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
