package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	util "github.com/kudobuilder/kudo/pkg/util/kudo"

	"github.com/spf13/afero"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpgradeCommand_Validation(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		instanceName string
		err          string
	}{
		{"no argument", []string{}, "instance", "expecting exactly one argument - name of the package or path to upgrade"},
		{"too many arguments", []string{"aaa", "bbb"}, "instance", "expecting exactly one argument - name of the package or path to upgrade"},
		{"no instance name", []string{"arg"}, "", "please use --instance and specify instance name. It cannot be empty"},
	}

	for _, tt := range tests {
		cmd := newUpgradeCmd(afero.NewOsFs())
		cmd.SetArgs(tt.args)
		if tt.instanceName != "" {
			cmd.Flags().Set("instance", tt.instanceName)
		}
		_, err := cmd.ExecuteC()
		if err.Error() != tt.err {
			t.Errorf("%s: expecting error %s got %v", tt.name, tt.err, err)
		}
	}
}

func newTestClient() *kudo.Client {
	return kudo.NewClientFromK8s(fake.NewSimpleClientset())
}

func TestUpgrade(t *testing.T) {
	testOv := v1alpha1.OperatorVersion{
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

	installNamespace := "default"
	tests := []struct {
		name               string
		newVersion         string
		instanceExists     bool
		ovExists           bool
		errMessageContains string
	}{
		{"instance does not exist", "1.1.1", false, true, "instance test in namespace default does not exist in the cluster"},
		{"operatorversion does not exist", "1.1.1", true, false, "no operator version for this operator installed yet"},
		{"upgrade to same version", "1.0", true, true, "upgraded version 1.0 is the same or smaller"},
		{"upgrade to smaller version", "0.1", true, true, "upgraded version 0.1 is the same or smaller"},
		{"upgrade to smaller version", "1.1.1", true, true, ""},
	}

	for _, tt := range tests {
		c := newTestClient()
		if tt.instanceExists {
			c.InstallInstanceObjToCluster(&testInstance, installNamespace)
		}
		if tt.ovExists {
			c.InstallOperatorVersionObjToCluster(&testOv, installNamespace)
		}
		newOv := testOv
		newOv.Spec.Version = tt.newVersion

		err := upgrade(&newOv, c, &options{
			InstanceName: "test",
			Namespace:    installNamespace,
		})
		if err != nil {
			if !strings.Contains(err.Error(), tt.errMessageContains) {
				t.Errorf("%s: expected error '%s' but got '%v'", tt.name, tt.errMessageContains, err)
			}
		} else if tt.errMessageContains != "" {
			t.Errorf("%s: expected no error but got %v", tt.name, err)
		} else {
			// the upgrade should have passed without error
			instance, err := c.GetInstance(testInstance.Name, installNamespace)
			if err != nil {
				t.Errorf("%s: error when getting instance to verify the test: %v", tt.name, err)
			}
			expectedVersion := fmt.Sprintf("test-%s", tt.newVersion)
			if instance.Spec.OperatorVersion.Name != expectedVersion {
				t.Errorf("%s: instance has wrong version '%s', expected '%s'", tt.name, instance.Spec.OperatorVersion.Name, expectedVersion)
			}
		}
	}
}
