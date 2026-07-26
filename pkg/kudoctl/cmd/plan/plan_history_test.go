package plan

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

func TestPlanHistory(t *testing.T) {
	instance := &kudoapi.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Status: kudoapi.InstanceStatus{
			PlanStatus: map[string]kudoapi.PlanStatus{
				"deploy": {
					Name:   "deploy",
					Status: kudoapi.ExecutionComplete,
				},
			},
		},
	}

	var tests = []struct {
		name            string
		instance        *kudoapi.Instance
		instanceNameArg string
		errorMessage    string
		expectedOutput  string
	}{
		{name: "nonexisting instance", instanceNameArg: "nonexisting", errorMessage: "instance default/nonexisting does not exist"},
		{name: "existing instance", instance: instance, instanceNameArg: "test", expectedOutput: ".\n└── deploy (COMPLETE)\n\n"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			kc := kudo.NewClientFromK8s(fake.NewSimpleClientset(), kubefake.NewSimpleClientset())
			if tt.instance != nil {
				_, err := kc.InstallInstanceObjToCluster(tt.instance, "default")
				if err != nil {
					t.Fatalf("%s: error when setting up a test - %v", tt.name, err)
				}
			}

			err := planHistory(kc, &Options{Out: &buf, Instance: tt.instanceNameArg}, "default")
			if tt.errorMessage != "" {
				assert.EqualError(t, err, tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}
			if tt.expectedOutput != "" {
				assert.Equal(t, tt.expectedOutput, buf.String())
			}
		})
	}
}
