package cmd

import (
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	util "github.com/kudobuilder/kudo/pkg/util/kudo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestClient() *kudo.Client {
	return kudo.NewClientFromK8s(fake.NewSimpleClientset())
}

func TestUninstall(t *testing.T) {
	testInstance := v1alpha1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1alpha1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
				util.OperatorLabel:        "test",
			},
			Name: "test",
		},
		Spec: v1alpha1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "test-1.0",
			},
		},
	}

	settings := env.DefaultSettings

	kc := newTestClient()
	_, err := kc.InstallInstanceObjToCluster(&testInstance, settings.Namespace)
	if err != nil {
		t.Fatalf("failed to install instance: %v", err)
	}

	cmd := uninstallCmd{}
	err = cmd.uninstall(kc, "nonexisting-instance", settings)
	if err == nil {
		t.Errorf("expected an error but got none")
	}

	errMsg := "instance nonexisting-instance in namespace default does not exist in the cluster"
	if err.Error() != errMsg {
		t.Errorf("expected error message '%s' but got '%v'", errMsg, err)
	}

	err = cmd.uninstall(kc, testInstance.Name, settings)
	if err != nil {
		t.Errorf("failed to uninstall instance: %v", err)
	}

	instance, err := kc.GetInstance(testInstance.Name, settings.Namespace)
	if err != nil {
		t.Errorf("failed to get instance: %v", err)
	}

	if instance != nil {
		t.Errorf("instance %s still found after deletion", testInstance.Name)
	}
}
