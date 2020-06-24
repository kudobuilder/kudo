package upgrade

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	engtask "github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	util "github.com/kudobuilder/kudo/pkg/util/kudo"
)

const (
	installNamespace = "default"
)

func Test_UpgradeOperatorVersion(t *testing.T) {
	testO := v1beta1.Operator{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Operator",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}

	testOv := v1beta1.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-1.0",
		},
		Spec: v1beta1.OperatorVersionSpec{
			Version: "1.0",
			Operator: v1.ObjectReference{
				Name: "test",
			},
		},
	}

	testInstance := v1beta1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				util.OperatorLabel: "test",
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
		name               string
		newVersion         string
		instanceExists     bool
		ovExists           bool
		errMessageContains string
	}{
		{"instance does not exist", "1.1.1", false, true, "instance default/test does not exist in the cluster"},
		{"operatorversion does not exist", "1.1.1", true, false, "operatorversion default/test-1.0 does not exist in the cluster"},
		{"upgrade to same version", "1.0", true, true, "upgraded version 1.0 is the same or smaller"},
		{"upgrade to smaller version", "0.1", true, true, "upgraded version 0.1 is the same or smaller"},
		{"upgrade to smaller version", "1.1.1", true, true, ""},
	}

	for _, tt := range tests {
		c := kudo.NewClientFromK8s(fake.NewSimpleClientset(), kubefake.NewSimpleClientset())
		if _, err := c.InstallOperatorObjToCluster(&testO, installNamespace); err != nil {
			t.Errorf("%s: failed to install operator: %v", tt.name, err)
		}

		if tt.instanceExists {
			if _, err := c.InstallInstanceObjToCluster(&testInstance, installNamespace); err != nil {
				t.Errorf("%s: failed to install instance: %v", tt.name, err)
			}
		}
		if tt.ovExists {
			if _, err := c.InstallOperatorVersionObjToCluster(&testOv, installNamespace); err != nil {
				t.Errorf("%s: failed to install operator version: %v", tt.name, err)
			}
		}
		newOv := testOv
		newOv.Spec.Version = tt.newVersion
		newOv.SetNamespace(installNamespace)

		err := OperatorVersion(c, &newOv, "test", nil, nil)
		if err != nil {
			if !strings.Contains(err.Error(), tt.errMessageContains) {
				t.Errorf("%s: expected error '%s' but got '%v'", tt.name, tt.errMessageContains, err)
			}
		} else if tt.errMessageContains != "" {
			t.Errorf("%s: expected no error but got %v", tt.name, err)
		} else {
			// the upgrade should have passed without error
			instance, err := c.GetInstance(testInstance.Name, installNamespace)
			if err != nil {
				t.Errorf("%s: error when getting instance to verify the test: %v", tt.name, err)
			}
			expectedVersion := fmt.Sprintf("test-%s", tt.newVersion)
			if instance.Spec.OperatorVersion.Name != expectedVersion {
				t.Errorf("%s: instance has wrong version '%s', expected '%s'", tt.name, instance.Spec.OperatorVersion.Name, expectedVersion)
			}
		}
	}
}

type testResolver struct {
	pkg packages.Package
}

func (r *testResolver) Resolve(name string, appVersion string, operatorVersion string) (*packages.Package, error) {
	return &r.pkg, nil
}

func Test_UpgradeOperatorVersionWithDependency(t *testing.T) {
	testO := v1beta1.Operator{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Operator",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}

	testOv := v1beta1.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-1.0",
		},
		Spec: v1beta1.OperatorVersionSpec{
			Version: "1.0",
			Operator: v1.ObjectReference{
				Name: "test",
			},
		},
	}

	testInstance := v1beta1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				util.OperatorLabel: "test",
			},
			Name: "test",
		},
		Spec: v1beta1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "test-1.0",
			},
		},
	}

	testDependency := packages.Package{
		Resources: &packages.Resources{
			Operator: &v1beta1.Operator{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dependency",
				},
			},
			OperatorVersion: &v1beta1.OperatorVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dependency-1.0",
				},
			},
		},
	}

	c := kudo.NewClientFromK8s(fake.NewSimpleClientset(), kubefake.NewSimpleClientset())

	_, err := c.InstallInstanceObjToCluster(&testInstance, installNamespace)
	assert.NoError(t, err)

	_, err = c.InstallOperatorVersionObjToCluster(&testOv, installNamespace)
	assert.NoError(t, err)

	_, err = c.InstallOperatorObjToCluster(&testO, installNamespace)
	assert.NoError(t, err)

	newOv := testOv
	newOv.Name = "test-1.1"
	newOv.Spec.Version = "1.1"
	newOv.Spec.Tasks = append(newOv.Spec.Tasks, v1beta1.Task{
		Name: "dependency",
		Kind: engtask.KudoOperatorTaskKind,
		Spec: v1beta1.TaskSpec{
			KudoOperatorTaskSpec: v1beta1.KudoOperatorTaskSpec{
				Package: "dependency",
			},
		},
	})
	newOv.SetNamespace(installNamespace)

	resolver := &testResolver{testDependency}

	err = OperatorVersion(c, &newOv, "test", nil, resolver)
	assert.NoError(t, err)

	assert.True(t, c.OperatorExistsInCluster("dependency", "default"))

	depOv, err := c.GetOperatorVersion("dependency-1.0", "default")
	assert.NoError(t, err)
	assert.NotNil(t, depOv)
}
