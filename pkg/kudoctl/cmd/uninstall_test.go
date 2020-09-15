package cmd

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	util "github.com/kudobuilder/kudo/pkg/util/kudo"
)

func newTestClient() *kudo.Client {
	return kudo.NewClientFromK8s(fake.NewSimpleClientset(), kubefake.NewSimpleClientset())
}

func TestUninstall(t *testing.T) {
	testInstance := kudoapi.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				util.OperatorLabel: "test",
			},
			Name: "test",
		},
		Spec: kudoapi.InstanceSpec{
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

	options := uninstallOptions{
		InstanceName: "nonexisting-instance",
	}

	cmd := uninstallCmd{}
	err = cmd.uninstall(kc, options, settings)
	if err == nil {
		t.Errorf("expected an error but got none")
	}

	errMsg := "instance nonexisting-instance in namespace default does not exist in the cluster"
	if err.Error() != errMsg {
		t.Errorf("expected error message '%s' but got '%v'", errMsg, err)
	}

	options.InstanceName = testInstance.Name
	options.Wait = true

	err = cmd.uninstall(kc, options, settings)
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
