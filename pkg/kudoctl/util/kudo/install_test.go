package kudo

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/util/convert"
)

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
		kc := NewClientFromK8s(client)

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

		err := InstallPackage(kc, &testResources, tt.skipInstance, "", namespace, tt.installParameters, false)
		if tt.err != "" {
			assert.ErrorContains(t, err, tt.err)
		}
	}
}
