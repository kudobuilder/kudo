package plan

import (
	"bytes"
	"flag"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/output"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

var (
	testTime     = time.Date(2019, 10, 17, 1, 1, 1, 1, time.UTC)
	updateGolden = flag.Bool("update", false, "update .golden files")
)

func TestStatus(t *testing.T) {
	ov := &kudoapi.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-1.0",
		},
		Spec: kudoapi.OperatorVersionSpec{
			Version: "1.0",
			Plans: map[string]kudoapi.Plan{
				"zzzinvalid": {
					Phases: []kudoapi.Phase{
						{
							Name: "zzzinvalid",
							Steps: []kudoapi.Step{
								{
									Name: "zzzinvalid",
								},
							},
						},
					},
				},
				"validate": {
					Phases: []kudoapi.Phase{
						{
							Name: "validate",
							Steps: []kudoapi.Step{
								{
									Name: "validate",
								},
							},
						},
					},
				},
				"deploy": {},
			},
		}}
	instance := &kudoapi.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: kudoapi.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "test-1.0",
			},
		},
	}

	fatalErrInstance := instance.DeepCopy()
	fatalErrInstance.Status = kudoapi.InstanceStatus{
		PlanStatus: map[string]kudoapi.PlanStatus{
			"deploy": {
				Name:                 "deploy",
				Status:               kudoapi.ExecutionFatalError,
				LastUpdatedTimestamp: &metav1.Time{Time: testTime},
				Phases: []kudoapi.PhaseStatus{
					{
						Name:   "deploy",
						Status: kudoapi.ExecutionFatalError,
						Steps: []kudoapi.StepStatus{
							{
								Name:    "deploy",
								Status:  kudoapi.ExecutionFatalError,
								Message: "error detail",
							},
						},
					},
				},
			},
		},
	}

	var tests = []struct {
		name            string
		instance        *kudoapi.Instance
		ov              *kudoapi.OperatorVersion
		instanceNameArg string
		errorMessage    string
		expectedOutput  string
		output          output.Type
		goldenFile      string
	}{
		{name: "nonexisting instance", instanceNameArg: "nonexisting", errorMessage: "instance default/nonexisting does not exist"},
		{name: "nonexisting ov", instance: instance, instanceNameArg: "test", errorMessage: "OperatorVersion test-1.0 from instance default/test does not exist"},
		{name: "no plan run", instance: instance, ov: ov, instanceNameArg: "test", expectedOutput: "No plan ever run for instance - nothing to show for instance test\n"},
		{name: "text output", instance: fatalErrInstance, ov: ov, instanceNameArg: "test", goldenFile: "planstatus.txt"},
		{name: "json output", instance: fatalErrInstance, ov: ov, instanceNameArg: "test", output: "json", goldenFile: "planstatus.json"},
		{name: "yaml output", instance: fatalErrInstance, ov: ov, instanceNameArg: "test", output: "yaml", goldenFile: "planstatus.yaml"},
		{name: "invalid output", instance: fatalErrInstance, ov: ov, instanceNameArg: "test", output: "invalid", errorMessage: output.InvalidOutputError},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			kc := kudo.NewClientFromK8s(fake.NewSimpleClientset(), kubefake.NewSimpleClientset())
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
			err := status(kc, &StatusOptions{Out: &buf, Instance: tt.instanceNameArg, Output: tt.output}, "default")
			if err != nil {
				assert.Equal(t, tt.errorMessage, err.Error())
			}
			if tt.goldenFile != "" {
				gp := filepath.Join("testdata", tt.goldenFile+".golden")

				if *updateGolden {
					t.Logf("updating golden file %s", tt.goldenFile)

					//nolint:gosec
					if err := ioutil.WriteFile(gp, buf.Bytes(), 0644); err != nil {
						t.Fatalf("failed to update golden file: %s", err)
					}
				}

				g, err := ioutil.ReadFile(gp)
				if err != nil {
					t.Fatalf("failed reading .golden: %s", err)
				}

				assert.Equal(t, string(g), buf.String(), "for golden file: %s, for test %s", gp, tt.name)
			}
			if tt.expectedOutput != "" {
				assert.Equal(t, buf.String(), tt.expectedOutput)
			}
		})
	}
}
