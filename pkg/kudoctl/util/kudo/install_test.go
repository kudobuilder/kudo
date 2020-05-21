package kudo

import (
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/util/convert"
)

var update = flag.Bool("update", false, "update .golden files")

func Test_InstallPackage(t *testing.T) {
	resources := packages.Resources{
		Operator: &v1beta1.Operator{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kudo.dev/v1beta1",
				Kind:       "Operator",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: v1beta1.OperatorSpec{
				KubernetesVersion: "1.15",
			},
		},
		Instance: &v1beta1.Instance{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kudo.dev/v1beta1",
				Kind:       "Instance",
			},
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"operator": "test",
				},
				Name: "test",
			},
			Spec: v1beta1.InstanceSpec{
				OperatorVersion: v1.ObjectReference{
					Name: "test-1.0",
				},
			},
		},
		OperatorVersion: &v1beta1.OperatorVersion{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kudo.dev/v1beta1",
				Kind:       "OperatorVersion",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-1.0", "operator"),
			},
			Spec: v1beta1.OperatorVersionSpec{
				Version: "1.0",
			},
		},
	}
	tv := true
	tests := []struct {
		name              string
		parameters        []v1beta1.Parameter
		installParameters map[string]string
		skipInstance      bool
		err               string
	}{
		{"all parameters with defaults", []v1beta1.Parameter{{Name: "param", Required: &tv, Default: convert.StringPtr("aaa")}}, map[string]string{}, false, ""},
		{"missing parameter provided", []v1beta1.Parameter{{Name: "param", Required: &tv}}, map[string]string{"param": "value"}, false, ""},
		{"missing parameter", []v1beta1.Parameter{{Name: "param", Required: &tv, Default: nil}}, map[string]string{}, false, "missing required parameters during installation: param"},
		{"multiple missing parameter", []v1beta1.Parameter{{Name: "param", Required: &tv}, {Name: "param2", Required: &tv}}, map[string]string{}, false, "missing required parameters during installation: param,param2"},
		{"skip instance ignores missing parameter", []v1beta1.Parameter{{Name: "param", Required: &tv}}, map[string]string{}, true, ""},
	}

	for _, tt := range tests {
		client := fake.NewSimpleClientset()
		kc := NewClientFromK8s(client, kubefake.NewSimpleClientset())

		fakeDiscovery, ok := client.Discovery().(*fakediscovery.FakeDiscovery)
		if !ok {
			t.Fatalf("couldn't convert Discovery() to *FakeDiscovery")
		}
		fakeDiscovery.FakedServerVersion = &version.Info{
			GitVersion: "v1.16.0",
		}

		testResources := resources
		testResources.OperatorVersion.Spec.Parameters = tt.parameters
		namespace := "default" //nolint:goconst

		err := InstallPackage(kc, &testResources, tt.skipInstance, "", namespace, tt.installParameters, false, false, 0)
		if tt.err != "" {
			assert.ErrorContains(t, err, tt.err)
		}
	}
}

func TestNamespaceManifestRendering(t *testing.T) {

	namespaceFile := "testdata/namespace.yaml"

	ns, err := ioutil.ReadFile(namespaceFile)
	tassert.NoError(t, err)

	params := make(map[string]string)
	params["foo"] = "bar-param"

	rendered, err := render("namespace", string(ns), testResources(), "foo-bar", "namespace-name", params)
	tassert.NoError(t, err)

	file := "rendered-namespace.yaml"
	gf := filepath.Join("testdata", file+".golden")

	if *update {
		t.Log("update golden file")
		if err := ioutil.WriteFile(gf, []byte(rendered), 0644); err != nil {
			t.Fatalf("failed to update golden file: %s", err)
		}
	}

	golden, err := ioutil.ReadFile(gf)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}

	assert.Equal(t, string(golden), string(rendered), "for golden file: %s", gf)
}

func testResources() *packages.Resources {
	result := &packages.Resources{
		Operator: &v1beta1.Operator{
			ObjectMeta: metav1.ObjectMeta{Name: "InstanceName"},
		},
		OperatorVersion: &v1beta1.OperatorVersion{
			Spec: v1beta1.OperatorVersionSpec{
				AppVersion: "1.0",
				Version:    "2.0",
			},
		},
		Instance: nil,
	}
	return result
}
