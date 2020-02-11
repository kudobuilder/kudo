package cmd

import (
	"strings"
	"testing"

	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	util "github.com/kudobuilder/kudo/pkg/util/kudo"
)

func TestUpdateCommand_Validation(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		instanceName string
		err          string
	}{
		{"too many arguments", []string{"aaa"}, "instance", "expecting no arguments provided"},
		{"no instance name", []string{}, "", "--instance flag has to be provided"},
		{"no parameter", []string{}, "instance", "need to specify at least one parameter to override "},
	}

	for _, tt := range tests {
		cmd := newUpdateCmd()
		cmd.SetArgs(tt.args)

		if tt.instanceName != "" {
			if err := cmd.Flags().Set("instance", tt.instanceName); err != nil {
				t.Fatal(err)
			}
		}
		_, err := cmd.ExecuteC()
		assert.ErrorContains(t, err, tt.err)
	}
}

func TestUpdate(t *testing.T) {
	testInstance := v1beta1.Instance{
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
		Spec: v1beta1.InstanceSpec{
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
		{"update arguments", true, map[string]string{"param": "value"}, ""},
	}

	for _, tt := range tests {
		c := newTestClient()
		if tt.instanceExists {
			if _, err := c.InstallInstanceObjToCluster(&testInstance, installNamespace); err != nil {
				t.Fatal(err)
			}
		}

		err := update(testInstance.Name, c, &updateOptions{Parameters: tt.parameters}, env.DefaultSettings)
		if err != nil {
			if !strings.Contains(err.Error(), tt.errMessageContains) {
				t.Errorf("%s: expected error '%s' but got '%v'", tt.name, tt.errMessageContains, err)
			}
		} else if tt.errMessageContains != "" {
			t.Errorf("%s: expected error '%s' but got nil", tt.name, tt.errMessageContains)
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
