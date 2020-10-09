package status

import (
	"context"
	"testing"

	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	label "github.com/kudobuilder/kudo/pkg/util/kudo"
)

func TestIsReady(t *testing.T) {
	instance := &kudoapi.Instance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "i",
			Namespace: "n",
		},
	}
	tests := []struct {
		name    string
		isReady bool
		objs    []runtime.Object
	}{
		{"no linked resources, ready", true, []runtime.Object{}},
		{"one ready deployment", true, []runtime.Object{readyDeployment()}},
		{"one not ready deployment", false, []runtime.Object{notReadyDeployment()}},
		{"one ready and one not ready deployment", false, []runtime.Object{readyDeployment(), notReadyDeployment()}},
	}
	for _, tt := range tests {
		c := fake.NewFakeClientWithScheme(scheme.Scheme)
		for _, obj := range tt.objs {
			err := c.Create(context.TODO(), obj)
			if err != nil {
				t.Errorf("Error in test setup for %s. %v", tt.name, err)
			}
		}
		ready, _, _ := IsReady(*instance, c)
		if ready != tt.isReady {
			t.Errorf("%s: expected instance to be ready: %t but got ready: %t", tt.name, tt.isReady, ready)
		}
	}
}

func readyDeployment() runtime.Object {
	var replicas int32 = 2
	return &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  "n",
			Name:       "foo",
			UID:        "8764ae47-9092-11e4-8393-42010af018ff",
			Generation: 1,
			Labels:     map[string]string{label.HeritageLabel: "kudo", label.InstanceLabel: "i"},
		},
		Spec: apps.DeploymentSpec{
			Replicas: &replicas,
		},
		Status: apps.DeploymentStatus{
			ObservedGeneration:  1,
			Replicas:            2,
			UpdatedReplicas:     2,
			AvailableReplicas:   2,
			UnavailableReplicas: 0,
		},
	}
}

func notReadyDeployment() runtime.Object {
	var replicas int32 = 2
	return &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  "n",
			Name:       "foo2",
			UID:        "8764ae47-9092-11e4-8393-42010af018ff",
			Generation: 1,
			Labels:     map[string]string{label.HeritageLabel: "kudo", label.InstanceLabel: "i"},
		},
		Spec: apps.DeploymentSpec{
			Replicas: &replicas,
		},
		Status: apps.DeploymentStatus{
			ObservedGeneration:  1,
			Replicas:            2,
			UpdatedReplicas:     2,
			AvailableReplicas:   1,
			UnavailableReplicas: 1,
		},
	}
}
