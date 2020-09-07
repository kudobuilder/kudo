package get

import (
	"bytes"
	"flag"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/output"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

var updateGolden = flag.Bool("update", false, "update .golden files")

func TestValidate(t *testing.T) {
	tests := []struct {
		arg []string
		err string
	}{
		{nil, "expecting exactly one argument - \"instances, operators, operatorversions or all\""},                                 // 1
		{[]string{"arg", "arg2"}, "expecting exactly one argument - \"instances, operators, operatorversions or all\""},             // 2
		{[]string{}, "expecting exactly one argument - \"instances, operators, operatorversions or all\""},                          // 3
		{[]string{"somethingelse"}, "expecting one of \"instances, operators, operatorversions or all\" and not \"somethingelse\""}, // 4
	}

	for _, tt := range tests {
		err := validate(tt.arg)
		assert.EqualError(t, err, tt.err)
	}
}

func newTestClient() *kudo.Client {
	return kudo.NewClientFromK8s(fake.NewSimpleClientset(), kubefake.NewSimpleClientset())
}

func TestGetInstances(t *testing.T) {
	testInstance := &kudoapi.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"operator": "test",
			},
			Name:      "test",
			Namespace: "default",
		},
		Spec: kudoapi.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "some-operator-0.1.0",
			},
		},
	}

	testOperator := &kudoapi.Operator{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Operator",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-operator",
			Namespace: "default",
		},
		Spec: kudoapi.OperatorSpec{
			Description: "A fancy Operator",
			KudoVersion: "0.16.0",
		},
	}

	testOperatorVersion := &kudoapi.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-operator-0.1.0",
			Namespace: "default",
		},
		Spec: kudoapi.OperatorVersionSpec{
			Operator: v1.ObjectReference{
				APIVersion: "kudo.dev/v1beta1",
				Kind:       "Operator",
				Name:       "some-operator",
			},
			Version: "0.1.0",
		},
	}

	kc := newTestClient()
	if _, err := kc.InstallInstanceObjToCluster(testInstance, "default"); err != nil {
		t.Fatal(err)
	}
	if _, err := kc.InstallOperatorObjToCluster(testOperator, "default"); err != nil {
		t.Fatal(err)
	}
	if _, err := kc.InstallOperatorVersionObjToCluster(testOperatorVersion, "default"); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		arg           string
		goldenFile    string
		output        output.Type
		expectedError string
	}{
		{name: "human readable instances", arg: "instances", goldenFile: "get-instances.txt", output: ""},
		{name: "yaml instances", arg: "instances", goldenFile: "get-instances.yaml", output: output.TypeYAML},
		{name: "json instances", arg: "instances", goldenFile: "get-instances.json", output: output.TypeJSON},
		{name: "human readable operators", arg: "operators", goldenFile: "get-operators.txt", output: ""},
		{name: "yaml operators", arg: "operators", goldenFile: "get-operators.yaml", output: output.TypeYAML},
		{name: "json operators", arg: "operators", goldenFile: "get-operators.json", output: output.TypeJSON},
		{name: "human readable operatorversions", arg: "operatorversions", goldenFile: "get-operatorversions.txt", output: ""},
		{name: "yaml operatorversions", arg: "operatorversions", goldenFile: "get-operatorversions.yaml", output: output.TypeYAML},
		{name: "json operatorversions", arg: "operatorversions", goldenFile: "get-operatorversions.json", output: output.TypeJSON},
		{name: "human readable all", arg: "all", goldenFile: "get-all.txt", output: ""},
		{name: "yaml all", arg: "all", goldenFile: "get-all.yaml", output: output.TypeYAML},
		{name: "json all", arg: "all", goldenFile: "get-all.json", output: output.TypeJSON},

		{name: "invalid output", arg: "instances", expectedError: output.InvalidOutputError, output: "invalid"},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			cmd := CmdOpts{Out: out, Output: tt.output, Namespace: "default", Client: kc}

			if err := Run([]string{tt.arg}, cmd); err != nil {
				if tt.expectedError != "" {
					assert.Equal(t, tt.expectedError, err.Error())
				} else {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if tt.goldenFile != "" {
				gp := filepath.Join("testdata", tt.goldenFile+".golden")

				if *updateGolden {
					t.Log("update golden file")

					//nolint:gosec
					if err := ioutil.WriteFile(gp, out.Bytes(), 0644); err != nil {
						t.Fatalf("failed to update golden file: %s", err)
					}
				}
				g, err := ioutil.ReadFile(gp)
				if err != nil {
					t.Fatalf("failed reading .golden: %s", err)
				}

				assert.Equal(t, string(g), out.String(), "output does not match .golden file %s", gp)
			}
		})
	}
}
