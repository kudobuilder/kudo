package cmd

import (
	"strings"
	"testing"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	util "github.com/kudobuilder/kudo/pkg/util/kudo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateCommand_Validation(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		instanceName string
		parameters   map[string]string
		err          string
	}{
		{"too many arguments", []string{"aaa"}, "instance", map[string]string{"param": "value"}, "expecting no arguments provided"},
		{"no instance name", []string{}, "", map[string]string{}, "--instance flag has to be provided"},
		{"no parameter", []string{}, "instance", map[string]string{}, "need to specify at least one parameter to override "},
	}

	for _, tt := range tests {
		cmd := newUpdateCmd()
		cmd.SetArgs(tt.args)
		for _, v := range tt.parameters {
			cmd.Flags().Set("p", v)
		}
		if tt.instanceName != "" {
			cmd.Flags().Set("instance", tt.instanceName)
		}
		_, err := cmd.ExecuteC()
		if !strings.Contains(err.Error(), tt.err) {
			t.Errorf("%s: expecting error %s got %v", tt.name, tt.err, err)
		}
	}
}

func TestUpdate(t *testing.T) {
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
		instanceExists     bool
		parameters         map[string]string
		errMessageContains string
	}{
		{"instance does not exist", false, map[string]string{"param": "value"}, "instance test in namespace default does not exist in the cluster"},
		{"update parameters", true, map[string]string{"param": "value"}, ""},
	}

	for _, tt := range tests {
		c := newTestClient()
		if tt.instanceExists {
			c.InstallInstanceObjToCluster(&testInstance, installNamespace)
		}

		err := update(testInstance.Name, c, &updateOptions{
			Namespace:  installNamespace,
			Parameters: tt.parameters,
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
			for k, v := range tt.parameters {
				value, ok := instance.Spec.Parameters[k]
				if !ok || value != v {
					t.Errorf("%s: expected parameter %s to be updated to %s but params are %v", tt.name, k, v, instance.Spec.Parameters)
				}
			}
		}
	}
}
