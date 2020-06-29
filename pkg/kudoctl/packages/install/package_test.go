package install

import (
	"flag"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
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
		kc := kudo.NewClientFromK8s(client, kubefake.NewSimpleClientset())

		fakeDiscovery, ok := client.Discovery().(*fakediscovery.FakeDiscovery)
		if !ok {
			t.Fatalf("couldn't convert Discovery() to *FakeDiscovery")
		}
		fakeDiscovery.FakedServerVersion = &version.Info{
			GitVersion: "v1.16.0",
		}

		testResources := resources
		testResources.OperatorVersion.Spec.Parameters = tt.parameters

		const namespace = "default"

		options := Options{
			SkipInstance: tt.skipInstance,
		}

		err := Package(kc, "", namespace, testResources, tt.installParameters, nil, options)
		if tt.err != "" {
			assert.EqualError(t, err, tt.err)
		}
	}
}

func testResources() packages.Resources {
	return packages.Resources{
		Operator: &v1beta1.Operator{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "OperatorName",
				Namespace: "default",
			},
		},
		OperatorVersion: &v1beta1.OperatorVersion{
			Spec: v1beta1.OperatorVersionSpec{
				AppVersion: "1.0",
				Version:    "2.0",
			},
		},
		Instance: &v1beta1.Instance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "InstanceName",
				Namespace: "default",
			},
		},
	}
}
