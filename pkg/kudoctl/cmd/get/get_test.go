package get

import (
	"testing"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		arg []string
		err string
	}{
		{nil, "expecting exactly one argument - \"instances\""},                          // 1
		{[]string{"arg", "arg2"}, "expecting exactly one argument - \"instances\""},      // 2
		{[]string{}, "expecting exactly one argument - \"instances\""},                   // 3
		{[]string{"somethingelse"}, "expecting \"instances\" and not \"somethingelse\""}, // 4
	}

	for _, tt := range tests {
		err := validate(tt.arg)
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("Expecting error message '%s' but got '%s'", tt.err, err)
			}
		}
	}
}

func newTestClient() *kudo.Client {
	return kudo.NewClientFromK8s(fake.NewSimpleClientset())
}

func TestGetInstances(t *testing.T) {
	testInstance := &v1alpha1.Instance{
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
	}
	tests := []struct {
		arg       []string
		err       string
		instances []string
	}{
		{nil, "expecting exactly one argument - \"instances\"", nil},                                   // 1
		{[]string{"arg", "arg2"}, "expecting exactly one argument - \"instances\"", nil},               // 2
		{[]string{}, "expecting exactly one argument - \"instances\"", nil},                            // 3
		{[]string{"somethingelse"}, "expecting \"instances\" and not \"somethingelse\"", nil},          // 4
		{[]string{"instances"}, "expecting \"instances\" and not \"somethingelse\"", []string{"test"}}, // 5
	}

	for i, tt := range tests {
		kc := newTestClient()
		kc.InstallInstanceObjToCluster(testInstance, "default")
		instanceList, err := getInstances(kc, env.DefaultSettings)
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d: Expecting error message '%s' but got '%s'", i+1, tt.err, err)
			}
		}
		missing := compareSlice(tt.instances, instanceList)
		for _, m := range missing {
			t.Errorf("%d: Missed expected instance \"%v\"", i+1, m)
		}
	}
}

func compareSlice(real, mock []string) []string {
	lm := len(mock)

	var diff []string

	for _, rv := range real {
		i := 0
		j := 0
		for _, mv := range mock {
			i++
			if rv == mv {
				continue
			}
			if rv != mv {
				j++
			}
			if lm <= j {
				diff = append(diff, rv)
			}
		}
	}
	return diff
}
