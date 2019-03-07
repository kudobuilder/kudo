package k8s

import (
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func newTestSimpleK2o() *K2oClient {
	client := K2oClient{}
	client.clientset = fake.NewSimpleClientset()
	return &client
}

func TestNewK2oClient(t *testing.T) {

	// For test case #1
	vars.KubeConfigPath = ""

	tests := []struct {
		err string
	}{
		{"invalid configuration: no configuration has been provided"}, // non existing test
	}

	for _, tt := range tests {
		// Just interested in errors
		_, err := NewK2oClient()
		if err.Error() != tt.err {
			t.Errorf("non existing test:\nexpected: %v\n     got: %v", tt.err, err.Error())
		}
	}
}

func TestK2oClient_CRDsInstalled(t *testing.T) {
	k2o := newTestSimpleK2o()
	err := k2o.CRDsInstalled()
	if err != nil {
		t.Errorf("\nexpected: <nil>\n     got: %v", err)
	}
}

func TestK2oClient_FrameworkExistsInCluster(t *testing.T) {

	obj := v1alpha1.Framework{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "Framework",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test",
		},
	}

	tests := []struct {
		bool     bool
		err      string
		createns string
		getns    string
		obj      *v1alpha1.Framework
	}{
		{false, "", "", "", nil},               // 1
		{false, "", "default", "default", nil}, // 2
		{true, "", "", "", &obj},               // 3
		{true, "", "default", "", &obj},        // 4
		{false, "", "", "kudo", &obj},          // 4
	}

	for i, tt := range tests {
		i := i
		k2o := newTestSimpleK2o()

		// create Framework
		_, err := k2o.clientset.KudoV1alpha1().Frameworks(tt.createns).Create(tt.obj)
		if err != nil {
			if err.Error() != "object does not implement the Object interfaces" {
				t.Errorf("unexpected error: %+v", err)
			}
		}

		// test if Framework exists in namespace
		vars.Namespace = tt.getns
		exist := k2o.FrameworkExistsInCluster("test")

		if tt.bool != exist {
			t.Errorf("%d:\nexpected: %v\n     got: %v", i+1, tt.bool, exist)
		}
	}
}

func TestK2oClient_AnyFrameworkVersionExistsInCluster(t *testing.T) {
	obj := v1alpha1.FrameworkVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "FrameworkVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test",
		},
	}

	tests := []struct {
		bool     bool
		err      string
		createns string
		getns    string
		obj      *v1alpha1.FrameworkVersion
	}{
		{false, "", "", "", nil},               // 1
		{false, "", "default", "default", nil}, // 2
		{true, "", "", "", &obj},               // 3
		{false, "", "", "qa", &obj},            // 4
		{true, "", "default", "", &obj},        // 5
	}

	for i, tt := range tests {
		i := i
		k2o := newTestSimpleK2o()

		// create FrameworkVersion
		k2o.clientset.KudoV1alpha1().FrameworkVersions(tt.createns).Create(tt.obj)

		// test if FrameworkVersion exists in namespace
		vars.Namespace = tt.getns
		exist := k2o.AnyFrameworkVersionExistsInCluster("test")
		if tt.bool != exist {
			t.Errorf("%d:\nexpected: %v\n     got: %v", i+1, tt.bool, exist)
		}
	}
}

func TestK2oClient_AnyInstanceExistsInCluster(t *testing.T) {
	obj := v1alpha1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
				"framework":               "test",
			},
			Name: "test",
		},
		Spec: v1alpha1.InstanceSpec{
			FrameworkVersion: v1.ObjectReference{
				Name: "test-1.0",
			},
		},
	}

	wrongObj := v1alpha1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
				"framework":               "test",
			},
			Name: "test",
		},
		Spec: v1alpha1.InstanceSpec{
			FrameworkVersion: v1.ObjectReference{
				Name: "test-0.9",
			},
		},
	}

	tests := []struct {
		bool     bool
		err      string
		createns string
		getns    string
		obj      *v1alpha1.Instance
	}{
		{false, "", "", "", nil},               // 1
		{false, "", "default", "default", nil}, // 2
		{true, "", "", "", &obj},               // 3
		{true, "", "", "", &obj},               // 4
		{false, "", "", "qa", &obj},            // 5
		{true, "", "qa", "qa", &obj},           // 6
		{false, "", "kudo", "", &wrongObj},     // 7
	}

	for i, tt := range tests {
		i := i
		k2o := newTestSimpleK2o()

		// create Instance
		k2o.clientset.KudoV1alpha1().Instances(tt.createns).Create(tt.obj)

		// test if FrameworkVersion exists in namespace
		vars.Namespace = tt.getns
		exist := k2o.AnyInstanceExistsInCluster("test", "1.0")
		if tt.bool != exist {
			t.Errorf("%d:\nexpected: %v\n     got: %v", i+1, tt.bool, exist)
		}
	}
}

func TestK2oClient_FrameworkVersionInClusterOutOfSync(t *testing.T) {
	obj := v1alpha1.FrameworkVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "FrameworkVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test-1.0",
		},
		Spec: v1alpha1.FrameworkVersionSpec{
			Version: "1.0",
		},
	}

	outdatedObj := v1alpha1.FrameworkVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "FrameworkVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test-0.9",
		},
		Spec: v1alpha1.FrameworkVersionSpec{
			Version: "0.9",
		},
	}

	tests := []struct {
		bool     bool
		err      string
		createns string
		getns    string
		obj      *v1alpha1.FrameworkVersion
	}{
		{false, "", "", "", nil},                  // 1
		{false, "", "default", "default", nil},    // 2
		{true, "", "", "", &obj},                  // 3
		{true, "", "", "", &obj},                  // 4
		{false, "", "", "qa", &obj},               // 5
		{true, "", "qa", "qa", &obj},              // 6
		{false, "", "kudo", "kudo", &outdatedObj}, // 7
	}

	for i, tt := range tests {
		i := i
		k2o := newTestSimpleK2o()

		// create Instance
		k2o.clientset.KudoV1alpha1().FrameworkVersions(tt.createns).Create(tt.obj)

		// test if FrameworkVersion exists in namespace
		vars.Namespace = tt.getns
		exist := k2o.FrameworkVersionInClusterOutOfSync("test", "1.0")
		if tt.bool != exist {
			t.Errorf("%d:\nexpected: %v\n     got: %v", i+1, tt.bool, exist)
		}
	}
}

func TestK2oClient_InstallFrameworkYamlToCluster(t *testing.T) {
	obj := v1alpha1.Framework{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "Framework",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test",
		},
	}

	tests := []struct {
		name     string
		err      string
		createns string
		obj      *v1alpha1.Framework
	}{
		{"", "frameworks.kudo.k8s.io \"\" not found", "", nil},                // 1
		{"", "frameworks.kudo.k8s.io \"\" not found", "default", nil},         // 2
		{"", "frameworks.kudo.k8s.io \"\" not found", "kudo", nil},            // 3
		{"test2", "frameworks.kudo.k8s.io \"test2\" not found", "kudo", &obj}, // 4
		{"test", "", "kudo", &obj},                                            // 5
	}

	for i, tt := range tests {
		i := i
		k2o := newTestSimpleK2o()

		// create Framework
		k2o.clientset.KudoV1alpha1().Frameworks(tt.createns).Create(tt.obj)

		// test if Framework exists in namespace
		vars.Namespace = tt.createns
		k2o.InstallFrameworkYamlToCluster(tt.obj)

		_, err := k2o.clientset.KudoV1alpha1().Frameworks(tt.createns).Get(tt.name, metav1.GetOptions{})
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d:\nexpected error: %v\n     got error: %v", i+1, tt.err, err)
			}
		}
	}
}

func TestK2oClient_InstallFrameworkVersionYamlToCluster(t *testing.T) {
	obj := v1alpha1.FrameworkVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "FrameworkVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test",
		},
	}

	tests := []struct {
		name     string
		err      string
		createns string
		obj      *v1alpha1.FrameworkVersion
	}{
		{"", "frameworkversions.kudo.k8s.io \"\" not found", "", nil},                // 1
		{"", "frameworkversions.kudo.k8s.io \"\" not found", "default", nil},         // 2
		{"", "frameworkversions.kudo.k8s.io \"\" not found", "kudo", nil},            // 3
		{"test2", "frameworkversions.kudo.k8s.io \"test2\" not found", "kudo", &obj}, // 4
		{"test", "", "kudo", &obj}, // 5

	}

	for i, tt := range tests {
		i := i
		k2o := newTestSimpleK2o()

		// create Framework
		k2o.clientset.KudoV1alpha1().FrameworkVersions(tt.createns).Create(tt.obj)

		// test if Framework exists in namespace
		vars.Namespace = tt.createns
		k2o.InstallFrameworkVersionYamlToCluster(tt.obj)

		_, err := k2o.clientset.KudoV1alpha1().FrameworkVersions(tt.createns).Get(tt.name, metav1.GetOptions{})
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d:\nexpected error: %v\n     got error: %v", i+1, tt.err, err)
			}
		}
	}
}

func TestK2oClient_InstallInstanceYamlToCluster(t *testing.T) {
	obj := v1alpha1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.k8s.io/v1alpha1",
			Kind:       "FrameworkVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: "test",
		},
	}

	tests := []struct {
		name     string
		err      string
		createns string
		obj      *v1alpha1.Instance
	}{
		{"", "instances.kudo.k8s.io \"\" not found", "", nil},                // 1
		{"", "instances.kudo.k8s.io \"\" not found", "default", nil},         // 2
		{"", "instances.kudo.k8s.io \"\" not found", "kudo", nil},            // 3
		{"test2", "instances.kudo.k8s.io \"test2\" not found", "kudo", &obj}, // 4
		{"test", "", "kudo", &obj},                                           // 5

	}

	for i, tt := range tests {
		i := i
		k2o := newTestSimpleK2o()

		// create Framework
		k2o.clientset.KudoV1alpha1().Instances(tt.createns).Create(tt.obj)

		// test if Framework exists in namespace
		vars.Namespace = tt.createns
		k2o.InstallInstanceYamlToCluster(tt.obj)

		_, err := k2o.clientset.KudoV1alpha1().Instances(tt.createns).Get(tt.name, metav1.GetOptions{})
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d:\nexpected error: %v\n     got error: %v", i+1, tt.err, err)
			}
		}
	}
}
