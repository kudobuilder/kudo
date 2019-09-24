package cmd

import (
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	util "github.com/kudobuilder/kudo/pkg/util/kudo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	err = cmd.uninstall(kc, "nonexisting-instance", false, settings)
	if err == nil {
		t.Errorf("expected an error but got none")
	}

	errMsg := "instance nonexisting-instance in namespace default does not exist in the cluster"
	if err.Error() != errMsg {
		t.Errorf("expected error message '%s' but got '%v'", errMsg, err)
	}

	err = cmd.uninstall(kc, testInstance.Name, false, settings)
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

func TestPurge(t *testing.T) {
	testOperator := v1alpha1.Operator{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1alpha1",
			Kind:       "Operator",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test",
		},
	}
	testOperatorVersion := v1alpha1.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1alpha1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test-1.0",
		},
		Spec: v1alpha1.OperatorVersionSpec{
			Version: "1.0",
			Operator: v1.ObjectReference{
				Name: "test",
			},
		},
	}
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
	_, err := kc.InstallOperatorObjToCluster(&testOperator, settings.Namespace)
	if err != nil {
		t.Fatalf("failed to install operator: %v", err)
	}

	_, err = kc.InstallOperatorVersionObjToCluster(&testOperatorVersion, settings.Namespace)
	if err != nil {
		t.Fatalf("failed to install operatorversion: %v", err)
	}

	_, err = kc.InstallInstanceObjToCluster(&testInstance, settings.Namespace)
	if err != nil {
		t.Fatalf("failed to install instance: %v", err)
	}

	cmd := uninstallCmd{}
	err = cmd.uninstall(kc, testInstance.Name, true, settings)
	if err != nil {
		t.Errorf("expected no error but get '%v'", err)
	}

	operatorVersion, err := kc.GetOperatorVersion(testOperatorVersion.Name, settings.Namespace)
	if err != nil {
		t.Errorf("failed to get operatorversion: %v", err)
	}
	if operatorVersion != nil {
		t.Errorf("operatorversion %s still found after deletion", testOperatorVersion.Name)
	}

	operator, err := kc.GetOperator(testOperator.Name, settings.Namespace)
	if err != nil {
		t.Errorf("failed to get operator: %v", err)
	}
	if operator != nil {
		t.Errorf("operator %s still found after deletion", testOperator.Name)
	}
}

func TestPurgeDifferentNamespaces(t *testing.T) {
	testOperator := v1alpha1.Operator{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1alpha1",
			Kind:       "Operator",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test",
		},
	}
	operatorNs := "foo"

	testOperatorVersion := v1alpha1.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1alpha1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test-1.0",
		},
		Spec: v1alpha1.OperatorVersionSpec{
			Version: "1.0",
			Operator: v1.ObjectReference{
				Name:      "test",
				Namespace: operatorNs,
			},
		},
	}
	operatorVersionNs := "bar"

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
				Name:      "test-1.0",
				Namespace: operatorVersionNs,
			},
		},
	}

	settings := env.DefaultSettings

	kc := newTestClient()
	_, err := kc.InstallOperatorObjToCluster(&testOperator, operatorNs)
	if err != nil {
		t.Fatalf("failed to install operator: %v", err)
	}

	_, err = kc.InstallOperatorVersionObjToCluster(&testOperatorVersion, operatorVersionNs)
	if err != nil {
		t.Fatalf("failed to install operatorversion: %v", err)
	}

	_, err = kc.InstallInstanceObjToCluster(&testInstance, settings.Namespace)
	if err != nil {
		t.Fatalf("failed to install instance: %v", err)
	}

	cmd := uninstallCmd{}
	err = cmd.uninstall(kc, testInstance.Name, true, settings)
	if err != nil {
		t.Errorf("expected no error but get '%v'", err)
	}

	operatorVersion, err := kc.GetOperatorVersion(testOperatorVersion.Name, operatorVersionNs)
	if err != nil {
		t.Errorf("failed to get operatorversion: %v", err)
	}
	if operatorVersion != nil {
		t.Errorf("operatorversion %s still found after deletion", testOperatorVersion.Name)
	}

	operator, err := kc.GetOperator(testOperator.Name, operatorNs)
	if err != nil {
		t.Errorf("failed to get operator: %v", err)
	}
	if operator != nil {
		t.Errorf("operator %s still found after deletion", testOperator.Name)
	}
}
