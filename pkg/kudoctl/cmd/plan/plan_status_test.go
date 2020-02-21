package plan

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

var (
	testTime = time.Date(2019, 10, 17, 1, 1, 1, 1, time.UTC)
)

func TestStatus(t *testing.T) {
	ov := &v1beta1.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-1.0",
		},
		Spec: v1beta1.OperatorVersionSpec{
			Version: "1.0",
			Plans:   map[string]v1beta1.Plan{"deploy": {}},
		}}
	instance := &v1beta1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: v1beta1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "test-1.0",
			},
		},
	}

	fatalErrInstance := instance.DeepCopy()
	fatalErrInstance.Status = v1beta1.InstanceStatus{
		PlanStatus: map[string]v1beta1.PlanStatus{
			"deploy": {
				Name:            "deploy",
				Status:          v1beta1.ExecutionFatalError,
				LastFinishedRun: &metav1.Time{Time: testTime},
				Phases: []v1beta1.PhaseStatus{
					{
						Name:   "deploy",
						Status: v1beta1.ExecutionFatalError,
						Steps: []v1beta1.StepStatus{
							{
								Name:    "deploy",
								Status:  v1beta1.ExecutionFatalError,
								Message: "error detail",
							},
						},
					},
				},
			},
		},
		AggregatedStatus: v1beta1.AggregatedStatus{
			LastUpdated: &metav1.Time{Time: testTime},
		},
	}

	var tests = []struct {
		name            string
		instance        *v1beta1.Instance
		ov              *v1beta1.OperatorVersion
		instanceNameArg string
		errorMessage    string
		expectedOutput  string
	}{
		{"nonexisting instance", nil, nil, "nonexisting", "Instance default/nonexisting does not exist", ""},
		{"nonexisting ov", instance, nil, "test", "OperatorVersion test-1.0 from instance default/test does not exist", ""},
		{"no plan run", instance, ov, "test", "", "No plan ever run for instance - nothing to show for instance test\n"},
		{"fatal error in a plan", fatalErrInstance, ov, "test", "", `Plan(s) for "test" in namespace "default":
.
└── test (Operator-Version: "test-1.0" Active-Plan: "deploy", last updated: "2019-10-17 01:01:01")
    └── Plan deploy ( strategy) [FATAL_ERROR], last finished 2019-10-17 01:01:01
        └── Phase deploy ( strategy) [FATAL_ERROR]
            └── Step deploy [FATAL_ERROR] (error detail)

`},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		kc := kudo.NewClientFromK8s(fake.NewSimpleClientset())
		if tt.instance != nil {
			_, err := kc.InstallInstanceObjToCluster(tt.instance, "default")
			if err != nil {
				t.Errorf("%s: error when setting up a test - %v", tt.name, err)
			}
		}
		if tt.ov != nil {
			_, err := kc.InstallOperatorVersionObjToCluster(tt.ov, "default")
			if err != nil {
				t.Errorf("%s: error when setting up a test - %v", tt.name, err)
			}
		}
		err := status(kc, &Options{Out: &buf, Instance: tt.instanceNameArg}, "default")
		if err != nil {
			assert.Equal(t, err.Error(), tt.errorMessage)
		}
		assert.Equal(t, buf.String(), tt.expectedOutput)
	}
}
