package kudo

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/util/kudo"
	util "github.com/kudobuilder/kudo/pkg/util/kudo"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestSimpleK2o() *Client {
	return NewClientFromK8s(fake.NewSimpleClientset())
}

func TestNewK2oClient(t *testing.T) {
	tests := []struct {
		err string
	}{
		{"invalid configuration: no configuration has been provided"}, // non existing test
	}

	for _, tt := range tests {
		// Just interested in errors
		_, err := NewClient("default", "")
		if err.Error() != tt.err {
			t.Errorf("non existing test:\nexpected: %v\n     got: %v", tt.err, err.Error())
		}
	}
}

func TestKudoClient_OperatorExistsInCluster(t *testing.T) {

	obj := v1alpha1.Operator{
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
	}

	tests := []struct {
		bool     bool
		err      string
		createns string
		getns    string
		obj      *v1alpha1.Operator
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

		// create Operator
		_, err := k2o.clientset.KudoV1alpha1().Operators(tt.createns).Create(tt.obj)
		if err != nil {
			if err.Error() != "object does not implement the Object interfaces" {
				t.Errorf("unexpected error: %+v", err)
			}
		}

		// test if Operator exists in namespace
		exist := k2o.OperatorExistsInCluster("test", tt.getns)

		if tt.bool != exist {
			t.Errorf("%d:\nexpected: %v\n     got: %v", i+1, tt.bool, exist)
		}
	}
}

func TestKudoClient_InstanceExistsInCluster(t *testing.T) {
	obj := v1alpha1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1alpha1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
				kudo.OperatorLabel:        "test",
			},
			Name: "test",
		},
		Spec: v1alpha1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "test-1.0",
			},
		},
	}

	wrongObj := v1alpha1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1alpha1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
				kudo.OperatorLabel:        "test",
			},
			Name: "test",
		},
		Spec: v1alpha1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "test-0.9",
			},
		},
	}

	instanceNamespace := "testnamespace"

	tests := []struct {
		name           string
		instanceExists bool
		namespace      string
		instanceName   string
		obj            *v1alpha1.Instance
	}{
		{"no existing instance in cluster", false, "", "", nil},                                                     // 1
		{"same namespace and instance name", true, instanceNamespace, obj.ObjectMeta.Name, &obj},                    // 3
		{"instance with new name", false, instanceNamespace, "nonexisting-instance-name", &obj},                     // 5
		{"same instance name in different namespace", false, "different-namespace", obj.ObjectMeta.Name, &wrongObj}, // 7
	}

	for _, tt := range tests {
		k2o := newTestSimpleK2o()

		// create Instance
		if tt.obj != nil {
			_, err := k2o.clientset.KudoV1alpha1().Instances(instanceNamespace).Create(tt.obj)
			if err != nil {
				t.Fatalf("%s: Error during test setup, cannot create test instance %v", tt.name, err)
			}

		}

		// test if OperatorVersion exists in namespace
		exist, _ := k2o.InstanceExistsInCluster("test", tt.namespace, "1.0", tt.instanceName)
		if tt.instanceExists != exist {
			t.Errorf("%s:\nexpected: %v\n     got: %v", tt.name, tt.instanceExists, exist)
		}
	}
}

func TestKudoClient_ListInstances(t *testing.T) {
	obj := v1alpha1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1alpha1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
				kudo.OperatorLabel:        "test",
			},
			Name: "test",
		},
		Spec: v1alpha1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "test-1.0",
			},
		},
	}

	installNamespace := "default"
	tests := []struct {
		expectedInstances []string
		namespace         string
		obj               *v1alpha1.Instance
	}{
		{[]string{}, installNamespace, nil},          // 1
		{[]string{obj.Name}, installNamespace, &obj}, // 2
		{[]string{}, "otherns", &obj},                // 3
	}

	for i, tt := range tests {
		k2o := newTestSimpleK2o()

		// create Instance
		if tt.obj != nil {
			_, err := k2o.clientset.KudoV1alpha1().Instances(installNamespace).Create(tt.obj)
			if err != nil {
				t.Errorf("%d: Error creating instance in tests setup", i+1)
			}
		}

		// test if OperatorVersion exists in namespace
		existingInstances, _ := k2o.ListInstances(tt.namespace)
		if !reflect.DeepEqual(tt.expectedInstances, existingInstances) {
			t.Errorf("%d:\nexpected: %v\n     got: %v", i+1, tt.expectedInstances, existingInstances)
		}
	}
}

func TestKudoClient_OperatorVersionsInstalled(t *testing.T) {
	operatorName := "test"
	obj := v1alpha1.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1alpha1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: fmt.Sprintf("%s-1.0", operatorName),
		},
		Spec: v1alpha1.OperatorVersionSpec{
			Version: "1.0",
		},
	}

	installNamespace := "default"
	tests := []struct {
		name             string
		expectedVersions []string
		namespace        string
		obj              *v1alpha1.OperatorVersion
	}{
		{"no operator version defined", []string{}, installNamespace, nil},
		{"operator version exists in the same namespace", []string{obj.Spec.Version}, installNamespace, &obj},
		{"operator version exists in different namespace", []string{}, "otherns", &obj},
	}

	for _, tt := range tests {
		k2o := newTestSimpleK2o()

		// create Instance
		if tt.obj != nil {
			_, err := k2o.clientset.KudoV1alpha1().OperatorVersions(installNamespace).Create(tt.obj)
			if err != nil {
				t.Errorf("Error creating operator version in tests setup for %s", tt.name)
			}
		}

		// test if OperatorVersion exists in namespace
		existingVersions, _ := k2o.OperatorVersionsInstalled(operatorName, tt.namespace)
		if !reflect.DeepEqual(tt.expectedVersions, existingVersions) {
			t.Errorf("%s:\nexpected: %v\n     got: %v", tt.name, tt.expectedVersions, existingVersions)
		}
	}
}

func TestKudoClient_InstallOperatorObjToCluster(t *testing.T) {
	obj := v1alpha1.Operator{
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
	}

	tests := []struct {
		name     string
		err      string
		createns string
		obj      *v1alpha1.Operator
	}{
		{"", "operators.kudo.dev \"\" not found", "", nil},                // 1
		{"", "operators.kudo.dev \"\" not found", "default", nil},         // 2
		{"", "operators.kudo.dev \"\" not found", "kudo", nil},            // 3
		{"test2", "operators.kudo.dev \"test2\" not found", "kudo", &obj}, // 4
		{"test", "", "kudo", &obj},                                        // 5
	}

	for i, tt := range tests {
		i := i
		k2o := newTestSimpleK2o()

		// create Operator
		k2o.clientset.KudoV1alpha1().Operators(tt.createns).Create(tt.obj)

		// test if Operator exists in namespace
		k2o.InstallOperatorObjToCluster(tt.obj, tt.createns)

		_, err := k2o.clientset.KudoV1alpha1().Operators(tt.createns).Get(tt.name, metav1.GetOptions{})
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d:\nexpected error: %v\n     got error: %v", i+1, tt.err, err)
			}
		}
	}
}

func TestKudoClient_InstallOperatorVersionObjToCluster(t *testing.T) {
	obj := v1alpha1.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1alpha1",
			Kind:       "OperatorVersion",
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
		obj      *v1alpha1.OperatorVersion
	}{
		{"", "operatorversions.kudo.dev \"\" not found", "", nil},                // 1
		{"", "operatorversions.kudo.dev \"\" not found", "default", nil},         // 2
		{"", "operatorversions.kudo.dev \"\" not found", "kudo", nil},            // 3
		{"test2", "operatorversions.kudo.dev \"test2\" not found", "kudo", &obj}, // 4
		{"test", "", "kudo", &obj},                                               // 5
	}

	for i, tt := range tests {
		i := i
		k2o := newTestSimpleK2o()

		// create Operator
		k2o.clientset.KudoV1alpha1().OperatorVersions(tt.createns).Create(tt.obj)

		// test if Operator exists in namespace
		k2o.InstallOperatorVersionObjToCluster(tt.obj, tt.createns)

		_, err := k2o.clientset.KudoV1alpha1().OperatorVersions(tt.createns).Get(tt.name, metav1.GetOptions{})
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d:\nexpected error: %v\n     got error: %v", i+1, tt.err, err)
			}
		}
	}
}

func TestKudoClient_InstallInstanceObjToCluster(t *testing.T) {
	obj := v1alpha1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1alpha1",
			Kind:       "OperatorVersion",
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
		{"", "instances.kudo.dev \"\" not found", "", nil},                // 1
		{"", "instances.kudo.dev \"\" not found", "default", nil},         // 2
		{"", "instances.kudo.dev \"\" not found", "kudo", nil},            // 3
		{"test2", "instances.kudo.dev \"test2\" not found", "kudo", &obj}, // 4
		{"test", "", "kudo", &obj},                                        // 5
	}

	for i, tt := range tests {
		i := i
		k2o := newTestSimpleK2o()

		// create Operator
		k2o.clientset.KudoV1alpha1().Instances(tt.createns).Create(tt.obj)

		// test if Operator exists in namespace
		k2o.InstallInstanceObjToCluster(tt.obj, tt.createns)

		_, err := k2o.clientset.KudoV1alpha1().Instances(tt.createns).Get(tt.name, metav1.GetOptions{})
		if err != nil {
			if err.Error() != tt.err {
				t.Errorf("%d:\nexpected error: %v\n     got error: %v", i+1, tt.err, err)
			}
		}
	}
}

func TestKudoClient_GetInstance(t *testing.T) {
	testInstance := v1alpha1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1alpha1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
				kudo.OperatorLabel:        "test",
			},
			Name: "test",
		},
		Spec: v1alpha1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "test-1.0",
			},
		},
	}

	installNamespace := "default"
	tests := []struct {
		name             string
		found            bool
		namespaceToQuery string
		storedInstance   *v1alpha1.Instance
	}{
		{"no instance exists", false, installNamespace, nil},                        // 1
		{"instance exists", true, installNamespace, &testInstance},                  // 2
		{"instance exists in different namespace", false, "otherns", &testInstance}, // 3
	}

	for i, tt := range tests {
		k2o := newTestSimpleK2o()

		// create Instance
		if tt.storedInstance != nil {
			_, err := k2o.clientset.KudoV1alpha1().Instances(installNamespace).Create(tt.storedInstance)
			if err != nil {
				t.Errorf("%d: Error creating instance in tests setup", i+1)
			}
		}

		// test if Instance exists in namespace
		actual, _ := k2o.GetInstance(testInstance.Name, tt.namespaceToQuery)
		if (actual != nil) != tt.found {
			t.Errorf("%s:\nexpected to be found: %v\n     got: %v", tt.name, tt.found, actual)
		}
	}
}

func TestKudoClient_GetOperatorVersion(t *testing.T) {
	operatorName := "test"
	testOv := v1alpha1.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1alpha1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
			},
			Name: fmt.Sprintf("%s-1.0", operatorName),
		},
		Spec: v1alpha1.OperatorVersionSpec{
			Version: "1.0",
		},
	}

	installNamespace := "default"
	tests := []struct {
		name      string
		found     bool
		namespace string
		storedOv  *v1alpha1.OperatorVersion
	}{
		{"no operator version defined", false, installNamespace, nil},
		{"operator version exists in the same namespace", true, installNamespace, &testOv},
		{"operator version exists in different namespace", false, "otherns", &testOv},
	}

	for _, tt := range tests {
		k2o := newTestSimpleK2o()

		// create Instance
		if tt.storedOv != nil {
			_, err := k2o.clientset.KudoV1alpha1().OperatorVersions(installNamespace).Create(tt.storedOv)
			if err != nil {
				t.Errorf("Error creating operator version in tests setup for %s", tt.name)
			}
		}

		// get OV by name and namespace
		actual, _ := k2o.GetOperatorVersion(testOv.Name, tt.namespace)
		if actual != nil != tt.found {
			t.Errorf("%s:\nexpected to be found: %v\n     got: %v", tt.name, tt.found, actual)
		}
	}
}

func TestKudoClient_UpdateOperatorVersion(t *testing.T) {
	testInstance := v1alpha1.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1alpha1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"controller-tools.k8s.io": "1.0",
				kudo.OperatorLabel:        "test",
			},
			Name: "test",
		},
		Spec: v1alpha1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "test-1.0",
			},
		},
	}

	installNamespace := "default"
	tests := []struct {
		name               string
		patchToVersion     *string
		existingParameters map[string]string
		parametersToPatch  map[string]string
		namespace          string
	}{
		{"patch to version", util.String("test-1.1.1"), nil, nil, installNamespace},
		{"patch adding new parameter", util.String("test-1.1.1"), nil, map[string]string{"param": "value"}, installNamespace},
		{"patch updating parameter", util.String("test-1.1.1"), map[string]string{"param": "value"}, map[string]string{"param": "value2"}, installNamespace},
		{"do not patch the version", nil, map[string]string{"param": "value"}, map[string]string{"param": "value2"}, installNamespace},
		{"patch with existing parameter should not override", util.String("1.1.1"), map[string]string{"param": "value"}, map[string]string{"other": "value2"}, installNamespace},
	}

	for _, tt := range tests {
		k2o := newTestSimpleK2o()

		// create Instance
		instanceToCreate := testInstance
		instanceToCreate.Spec.Parameters = tt.existingParameters
		_, err := k2o.clientset.KudoV1alpha1().Instances(installNamespace).Create(&instanceToCreate)
		if err != nil {
			t.Errorf("Error creating operator version in tests setup for %s", tt.name)
		}

		err = k2o.UpdateInstance(testInstance.Name, installNamespace, tt.patchToVersion, tt.parametersToPatch)
		instance, _ := k2o.GetInstance(testInstance.Name, installNamespace)
		if tt.patchToVersion != nil {
			if err != nil || instance.Spec.OperatorVersion.Name != util.StringValue(tt.patchToVersion) {
				t.Errorf("%s:\nexpected version: %v\n     got: %v, err: %v", tt.name, util.StringValue(tt.patchToVersion), instance.Spec.OperatorVersion.Name, err)
			}
		} else {
			if instance.Spec.OperatorVersion.Name != testInstance.Spec.OperatorVersion.Name {
				t.Errorf("%s:\nexpected version to not change from: %v\n err: %v", tt.name, instance.Spec.OperatorVersion.Name, err)
			}
		}

		// verify that parameters were updated
		for n, v := range tt.parametersToPatch {
			found, ok := instance.Spec.Parameters[n]
			if !ok || found != v {
				t.Errorf("%s: Value of parameter %s should have been updated to %s but is %s", tt.name, n, v, found)
			}
		}

		// make sure that we did not change parameters that should not have been updated
		for n, v := range tt.existingParameters {
			if _, ok := tt.parametersToPatch[n]; ok {
				continue
			}
			found, ok := instance.Spec.Parameters[n]
			fmt.Println(n)
			if !ok || found != v {
				t.Errorf("%s: Value of parameter %s should have not been updated from value %s but is %s", tt.name, n, v, found)
			}
		}
	}
}
