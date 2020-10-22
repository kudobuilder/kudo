package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
)

func TestIsHealthy(t *testing.T) {
	// this table skips resources covered by 'polymorphichelpers' and older APIs
	// for which the same code is used.
	tests := []struct {
		name    string
		input   runtime.Object
		healthy bool
		msg     string
	}{
		{
			name:    "unknown object type",
			input:   &corev1.ConfigMap{},
			healthy: true,
			msg:     "unknown type *v1.ConfigMap is marked healthy by default",
		},
		{
			name: "unhealthy CRD",
			input: &apiextv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Status: apiextv1.CustomResourceDefinitionStatus{
					Conditions: []apiextv1.CustomResourceDefinitionCondition{
						{
							Type:   apiextv1.Established,
							Status: apiextv1.ConditionFalse,
						},
					},
				},
			},
			healthy: false,
			msg:     "CRD foo is not healthy ( Conditions: [{Established False 0001-01-01 00:00:00 +0000 UTC  }] )",
		},
		{
			name: "healthy CRD",
			input: &apiextv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Status: apiextv1.CustomResourceDefinitionStatus{
					Conditions: []apiextv1.CustomResourceDefinitionCondition{
						{
							Type:   apiextv1.Established,
							Status: apiextv1.ConditionTrue,
						},
					},
				},
			},
			healthy: true,
			msg:     "CRD foo is now healthy",
		},
		{
			name: "unhealthy Job",
			input: &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Status: batchv1.JobStatus{
					Succeeded: 0,
				},
			},
			healthy: false,
			msg:     "job \"foo\" still running or failed",
		},
		{
			name: "healthy Job",
			input: &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Status: batchv1.JobStatus{
					Succeeded: 1,
				},
			},
			healthy: true,
			msg:     "job \"foo\" is marked healthy",
		},
		{
			name: "unhealthy Instance",
			input: &kudoapi.Instance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: kudoapi.InstanceSpec{
					PlanExecution: kudoapi.PlanExecution{
						PlanName: "deploy",
						Status:   kudoapi.ExecutionInProgress,
					},
				},
			},
			healthy: false,
			msg:     "instance default/foo active plan is in state IN_PROGRESS",
		},
		{
			name: "healthy Instance",
			input: &kudoapi.Instance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Spec: kudoapi.InstanceSpec{
					PlanExecution: kudoapi.PlanExecution{
						PlanName: "",
					},
				},
			},
			healthy: true,
			msg:     "instance default/foo is marked healthy",
		},
		{
			name: "unhealthy Pod",
			input: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodUnknown,
					Conditions: []corev1.PodCondition{
						{
							Type: corev1.PodReady,
						},
					},
				},
			},
			healthy: false,
			msg:     "pod default/foo is not running yet: Unknown",
		},
		{
			name: "healthy Pod",
			input: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			healthy: true,
			msg:     "",
		},
		{
			name: "unhealthy Namespace",
			input: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceTerminating,
				},
			},
			healthy: false,
			msg:     "namespace default is not active: Terminating",
		},
		{
			name: "healthy Namespace",
			input: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
				Status: corev1.NamespaceStatus{
					Phase: corev1.NamespaceActive,
				},
			},
			healthy: true,
			msg:     "",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			healthy, msg, _ := IsHealthy(test.input)

			assert.Equal(t, test.healthy, healthy)
			assert.Equal(t, test.msg, msg)
		})
	}
}
