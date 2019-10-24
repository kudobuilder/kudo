package install

import (
	"fmt"
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	util "github.com/kudobuilder/kudo/pkg/util/kudo"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
)

func TestValidate(t *testing.T) {

	tests := []struct {
		arg []string
		err string
	}{
		{nil, "expecting exactly one argument - name of the package or path to install"},                     // 1
		{[]string{"arg", "arg2"}, "expecting exactly one argument - name of the package or path to install"}, // 2
		{[]string{}, "expecting exactly one argument - name of the package or path to install"},              // 3
	}

	for _, tt := range tests {
		err := validate(tt.arg, DefaultOptions)
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("Expecting error message '%s' but got '%s'", tt.err, err)
			}
		}
	}
}

// TODO: adapt these tests to run as part of the 'resources' package instead.
func TestParameterValidation_InstallCrds(t *testing.T) {
	crds := packages.PackageCRDs{
		Operator: &v1alpha1.Operator{
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
			Spec: v1alpha1.OperatorSpec{
				KubernetesVersion: "1.15",
			},
		},
		Instance: &v1alpha1.Instance{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kudo.dev/v1alpha1",
				Kind:       "Instance",
			},
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"controller-tools.k8s.io": "1.0",
					"operator":                "test",
				},
				Name: "test",
			},
			Spec: v1alpha1.InstanceSpec{
				OperatorVersion: v1.ObjectReference{
					Name: "test-1.0",
				},
			},
		},
		OperatorVersion: &v1alpha1.OperatorVersion{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kudo.dev/v1alpha1",
				Kind:       "OperatorVersion",
			},
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"controller-tools.k8s.io": "1.0",
				},
				Name: fmt.Sprintf("%s-1.0", "operator"),
			},
			Spec: v1alpha1.OperatorVersionSpec{
				Version: "1.0",
			},
		},
	}

	tests := []struct {
		name              string
		parameters        []v1alpha1.Parameter
		installParameters map[string]string
		skipInstance      bool
		err               string
	}{
		{"all parameters with defaults", []v1alpha1.Parameter{{Name: "param", Required: true, Default: util.String("aaa")}}, map[string]string{}, false, ""},
		{"missing parameter provided", []v1alpha1.Parameter{{Name: "param", Required: true}}, map[string]string{"param": "value"}, false, ""},
		{"missing parameter", []v1alpha1.Parameter{{Name: "param", Required: true, Default: nil}}, map[string]string{}, false, "missing required parameters during installation: param"},
		{"multiple missing parameter", []v1alpha1.Parameter{{Name: "param", Required: true}, {Name: "param2", Required: true}}, map[string]string{}, false, "missing required parameters during installation: param,param2"},
		{"skip instance ignores missing parameter", []v1alpha1.Parameter{{Name: "param", Required: true}}, map[string]string{}, true, ""},
	}

	for _, tt := range tests {
		client := fake.NewSimpleClientset()
		kc := kudo.NewClientFromK8s(client)

		fakeDiscovery, ok := client.Discovery().(*fakediscovery.FakeDiscovery)
		if !ok {
			t.Fatalf("couldn't convert Discovery() to *FakeDiscovery")
		}
		fakeDiscovery.FakedServerVersion = &version.Info{
			GitVersion: "v1.16.0",
		}

		testCrds := crds
		testCrds.OperatorVersion.Spec.Parameters = tt.parameters
		options := &Options{}
		options.Parameters = tt.installParameters
		options.SkipInstance = tt.skipInstance

		err := installCrds(&testCrds, kc, options, env.DefaultSettings)
		if err != nil && err.Error() != tt.err {
			t.Errorf("%s: Expected error '%s', got '%s'", tt.name, tt.err, err.Error())
		}
	}
}
