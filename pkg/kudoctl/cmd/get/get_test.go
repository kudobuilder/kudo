package get

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
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
		assert.ErrorContains(t, err, tt.err)
	}
}

func newTestClient() *kudo.Client {
	return kudo.NewClientFromK8s(fake.NewSimpleClientset())
}

func TestGetInstances(t *testing.T) {
	testInstance := &v1beta1.Instance{
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
	}
	tests := []struct {
		instances []string
	}{
		{[]string{"test"}},
	}

	for _, tt := range tests {
		kc := newTestClient()
		if _, err := kc.InstallInstanceObjToCluster(testInstance, "default"); err != nil {
			t.Fatal(err)
		}
		instanceList, err := getInstances(kc, env.DefaultSettings)
		assert.NilError(t, err)
		tassert.EqualValues(t, tt.instances, instanceList, "missing instances")
	}
}
