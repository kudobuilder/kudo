package kudo

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/util/convert"
	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

const installNamespace = "default"

func newTestSimpleK2o() *Client {
	return NewClientFromK8s(fake.NewSimpleClientset(), kubefake.NewSimpleClientset())
}

func newFakeKubeClient() *kube.Client {
	client := kubefake.NewSimpleClientset()
	extClient := apiextensionfake.NewSimpleClientset()
	dynamicClient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())

	return &kube.Client{
		KubeClient:    client,
		ExtClient:     extClient,
		DynamicClient: dynamicClient}
}

func TestKudoClient_ValidateServedCrds(t *testing.T) {
	crdWrongVersion := apiextv1.CustomResourceDefinition{
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Versions: []apiextv1.CustomResourceDefinitionVersion{
				{
					Name:   "v1beta2",
					Served: true,
				},
			},
		},
		Status: apiextv1.CustomResourceDefinitionStatus{
			Conditions: []apiextv1.CustomResourceDefinitionCondition{
				{
					Type:   apiextv1.Established,
					Status: apiextv1.ConditionTrue,
				},
			},
		},
	}
	crdNotServed := apiextv1.CustomResourceDefinition{
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Versions: []apiextv1.CustomResourceDefinitionVersion{
				{
					Name:   "v1beta1",
					Served: false,
				},
				{
					Name:   "v1beta2",
					Served: true,
				},
			},
		},
		Status: apiextv1.CustomResourceDefinitionStatus{
			Conditions: []apiextv1.CustomResourceDefinitionCondition{
				{
					Type:   apiextv1.Established,
					Status: apiextv1.ConditionTrue,
				},
			},
		},
	}
	crdUnHealthy := apiextv1.CustomResourceDefinition{
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Versions: []apiextv1.CustomResourceDefinitionVersion{
				{
					Name:   "v1beta2",
					Served: true,
				},
			},
		},
		Status: apiextv1.CustomResourceDefinitionStatus{
			Conditions: []apiextv1.CustomResourceDefinitionCondition{
				{
					Type:   apiextv1.Established,
					Status: apiextv1.ConditionFalse,
				},
				{
					Type:   apiextv1.Terminating,
					Status: apiextv1.ConditionTrue,
				},
			},
		},
	}

	tests := []struct {
		name    string
		crdBase *apiextv1.CustomResourceDefinition
		err     string
	}{
		{name: "no crd", err: `CRDs invalid: CRD operators.kudo.dev is not installed       
CRD operatorversions.kudo.dev is not installed
CRD instances.kudo.dev is not installed       
`},
		{name: "wrong version", crdBase: &crdWrongVersion, err: `CRDs invalid: Expected API version v1beta1 was not found for operators.kudo.dev, api-server only supports [v1beta2]. Please update your KUDO CLI.       
Expected API version v1beta1 was not found for operatorversions.kudo.dev, api-server only supports [v1beta2]. Please update your KUDO CLI.
Expected API version v1beta1 was not found for instances.kudo.dev, api-server only supports [v1beta2]. Please update your KUDO CLI.       
`},
		{name: "not served", crdBase: &crdNotServed, err: `CRDs invalid: Expected API version v1beta1 for operators.kudo.dev is known to api-server, but is not served. Please update your KUDO CLI.       
Expected API version v1beta1 for operatorversions.kudo.dev is known to api-server, but is not served. Please update your KUDO CLI.
Expected API version v1beta1 for instances.kudo.dev is known to api-server, but is not served. Please update your KUDO CLI.       
`},
		{name: "unhealthy", crdBase: &crdUnHealthy, err: `CRDs invalid: CRD operators.kudo.dev is not healthy ( Conditions: [{Established False 0001-01-01 00:00:00 +0000 UTC  } {Terminating True 0001-01-01 00:00:00 +0000 UTC  }] )       
CRD operatorversions.kudo.dev is not healthy ( Conditions: [{Established False 0001-01-01 00:00:00 +0000 UTC  } {Terminating True 0001-01-01 00:00:00 +0000 UTC  }] )
CRD instances.kudo.dev is not healthy ( Conditions: [{Established False 0001-01-01 00:00:00 +0000 UTC  } {Terminating True 0001-01-01 00:00:00 +0000 UTC  }] )       
`},
	}

	crdNames := []string{"instances.kudo.dev", "operators.kudo.dev", "operatorversions.kudo.dev"}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			kudoClient := newTestSimpleK2o()
			kubeClient := newFakeKubeClient()

			if tt.crdBase != nil {
				for _, crdName := range crdNames {
					crd := tt.crdBase.DeepCopy()
					crd.Name = crdName
					_, _ = kubeClient.ExtClient.ApiextensionsV1().CustomResourceDefinitions().Create(context.TODO(), crd, metav1.CreateOptions{})
				}
			}

			err := kudoClient.VerifyServedCRDs(kubeClient)

			assert.EqualError(t, err, tt.err)

		})
	}

}

func TestKudoClient_OperatorExistsInCluster(t *testing.T) {

	obj := kudoapi.Operator{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Operator",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}

	tests := []struct {
		exists   bool
		err      string
		createns string
		getns    string
		obj      *kudoapi.Operator
	}{
		{},                                      // 1
		{createns: "default", getns: "default"}, // 2
		{exists: true, obj: &obj},               // 3
		{exists: true, createns: "default", obj: &obj, getns: "default"}, // 4
		{getns: "kudo", obj: &obj},                                       // 5
	}

	for i, tt := range tests {
		k2o := newTestSimpleK2o()

		// create Operator
		_, err := k2o.kudoClientset.
			KudoV1beta1().
			Operators(tt.createns).
			Create(context.TODO(), tt.obj, metav1.CreateOptions{})
		if err != nil {
			if err.Error() != "object does not implement the Object interfaces" {
				t.Errorf("unexpected error: %+v", err)
			}
		}

		// test if Operator exists in namespace
		exist := k2o.OperatorExistsInCluster("test", tt.getns)

		if tt.exists != exist {
			t.Errorf("%d:\nexpected: %v\n     got: %v", i+1, tt.exists, exist)
		}
	}
}

func TestKudoClient_InstanceExistsInCluster(t *testing.T) {
	obj := kudoapi.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				kudo.OperatorLabel: "test",
			},
			Name: "test",
		},
		Spec: kudoapi.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "test-1.0",
			},
		},
	}

	wrongObj := kudoapi.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				kudo.OperatorLabel: "test",
			},
			Name: "test",
		},
		Spec: kudoapi.InstanceSpec{
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
		obj            *kudoapi.Instance
	}{
		{name: "no existing instance in cluster"}, // 1
		{name: "same namespace and instance name", instanceExists: true, namespace: instanceNamespace, instanceName: obj.ObjectMeta.Name, obj: &obj}, // 3
		{name: "instance with new name", namespace: instanceNamespace, instanceName: "nonexisting-instance-name", obj: &obj},                         // 5
		{name: "same instance name in different namespace", namespace: "different-namespace", instanceName: obj.ObjectMeta.Name, obj: &wrongObj},     // 7
	}

	for _, tt := range tests {
		k2o := newTestSimpleK2o()

		// create Instance
		if tt.obj != nil {
			_, err := k2o.kudoClientset.
				KudoV1beta1().
				Instances(instanceNamespace).
				Create(context.TODO(), tt.obj, metav1.CreateOptions{})
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
	obj := kudoapi.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				kudo.OperatorLabel: "test",
			},
			Name: "test",
		},
		Spec: kudoapi.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "test-1.0",
			},
		},
	}

	tests := []struct {
		expectedInstances []runtime.Object
		namespace         string
		obj               *kudoapi.Instance
	}{
		{expectedInstances: []runtime.Object{}, namespace: installNamespace},                // 1
		{expectedInstances: []runtime.Object{&obj}, namespace: installNamespace, obj: &obj}, // 2
		{expectedInstances: []runtime.Object{}, namespace: "otherns", obj: &obj},            // 3
	}

	for i, tt := range tests {
		k2o := newTestSimpleK2o()

		// create Instance
		if tt.obj != nil {
			_, err := k2o.kudoClientset.
				KudoV1beta1().
				Instances(installNamespace).
				Create(context.TODO(), tt.obj, metav1.CreateOptions{})
			if err != nil {
				t.Errorf("%d: Error creating instance in tests setup", i+1)
			}
		}

		// test if OperatorVersion exists in namespace
		existingInstances, _ := k2o.ListInstancesAsRuntimeObject(tt.namespace)

		assert.Equal(t, len(tt.expectedInstances), len(existingInstances))

		metadataAccessor := meta.NewAccessor()
		for i := range existingInstances {
			expectedName, err := metadataAccessor.Name(tt.expectedInstances[i])
			assert.NoError(t, err)
			existingName, err := metadataAccessor.Name(existingInstances[i])
			assert.NoError(t, err)

			assert.Equal(t, expectedName, existingName)
		}
	}
}

func TestKudoClient_OperatorVersionsInstalled(t *testing.T) {
	operatorName := "test"
	obj := kudoapi.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-1.0", operatorName),
		},
		Spec: kudoapi.OperatorVersionSpec{
			Version: "1.0",
			Operator: v1.ObjectReference{
				Name: operatorName,
			},
		},
	}
	objWithSamePrefix := obj.DeepCopy()
	objWithSamePrefix.Name = operatorName + "-demo"
	objWithSamePrefix.Spec.Operator.Name = operatorName + "-demo"

	tests := []struct {
		name             string
		expectedVersions []string
		namespace        string
		obj              *kudoapi.OperatorVersion
	}{
		{name: "no operator version defined", expectedVersions: []string{}, namespace: installNamespace},
		{name: "operator version exists in the same namespace", expectedVersions: []string{obj.Spec.Version}, namespace: installNamespace, obj: &obj},
		{name: "operator version exists in different namespace", expectedVersions: []string{}, namespace: "otherns", obj: &obj},
		{name: "operator with same prefix exists", expectedVersions: []string{}, namespace: installNamespace, obj: objWithSamePrefix},
	}

	for _, tt := range tests {
		k2o := newTestSimpleK2o()

		// create Instance
		if tt.obj != nil {
			_, err := k2o.kudoClientset.
				KudoV1beta1().
				OperatorVersions(installNamespace).
				Create(context.TODO(), tt.obj, metav1.CreateOptions{})
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
	obj := kudoapi.Operator{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Operator",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}

	tests := []struct {
		name     string
		err      string
		createns string
		obj      *kudoapi.Operator
	}{
		{err: "operators.kudo.dev \"\" not found"},                                                  // 1
		{err: "operators.kudo.dev \"\" not found", createns: "default"},                             // 2
		{err: "operators.kudo.dev \"\" not found", createns: "kudo"},                                // 3
		{name: "test2", err: "operators.kudo.dev \"test2\" not found", createns: "kudo", obj: &obj}, // 4
		{name: "test", createns: "kudo", obj: &obj},                                                 // 5
	}

	for i, tt := range tests {
		k2o := newTestSimpleK2o()

		// create Operator
		k2o.kudoClientset.
			KudoV1beta1().
			Operators(tt.createns).
			Create(context.TODO(), tt.obj, metav1.CreateOptions{}) //nolint:errcheck

		// test if Operator exists in namespace
		k2o.InstallOperatorObjToCluster(tt.obj, tt.createns) //nolint:errcheck

		_, err := k2o.kudoClientset.
			KudoV1beta1().
			Operators(tt.createns).
			Get(context.TODO(), tt.name, metav1.GetOptions{})
		if tt.err != "" {
			assert.EqualError(t, err, tt.err, "failure in %v test case", i+1)
		}
	}
}

func TestKudoClient_InstallOperatorVersionObjToCluster(t *testing.T) {
	obj := kudoapi.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}

	tests := []struct {
		name     string
		err      string
		createns string
		obj      *kudoapi.OperatorVersion
	}{
		{err: "operatorversions.kudo.dev \"\" not found"},                                                  // 1
		{err: "operatorversions.kudo.dev \"\" not found", createns: "default"},                             // 2
		{err: "operatorversions.kudo.dev \"\" not found", createns: "kudo"},                                // 3
		{name: "test2", err: "operatorversions.kudo.dev \"test2\" not found", createns: "kudo", obj: &obj}, // 4
		{name: "test", createns: "kudo", obj: &obj},                                                        // 5
	}

	for i, tt := range tests {
		k2o := newTestSimpleK2o()

		// create Operator
		k2o.kudoClientset.
			KudoV1beta1().
			OperatorVersions(tt.createns).
			Create(context.TODO(), tt.obj, metav1.CreateOptions{}) //nolint:errcheck

		// test if Operator exists in namespace
		k2o.InstallOperatorVersionObjToCluster(tt.obj, tt.createns) //nolint:errcheck

		_, err := k2o.kudoClientset.
			KudoV1beta1().
			OperatorVersions(tt.createns).
			Get(context.TODO(), tt.name, metav1.GetOptions{})
		if tt.err != "" {
			assert.EqualError(t, err, tt.err, "failure in %v test case", i+1)
		}
	}
}

func TestKudoClient_InstallInstanceObjToCluster(t *testing.T) {
	obj := kudoapi.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}

	tests := []struct {
		name     string
		err      string
		createns string
		obj      *kudoapi.Instance
	}{
		{err: "instances.kudo.dev \"\" not found"},                                                  // 1
		{err: "instances.kudo.dev \"\" not found", createns: "default"},                             // 2
		{err: "instances.kudo.dev \"\" not found", createns: "kudo"},                                // 3
		{name: "test2", err: "instances.kudo.dev \"test2\" not found", createns: "kudo", obj: &obj}, // 4
		{name: "test", createns: "kudo", obj: &obj},                                                 // 5
	}

	for i, tt := range tests {
		k2o := newTestSimpleK2o()

		// create Operator
		k2o.kudoClientset.
			KudoV1beta1().
			Instances(tt.createns).
			Create(context.TODO(), tt.obj, metav1.CreateOptions{}) //nolint:errcheck

		// test if Operator exists in namespace
		k2o.InstallInstanceObjToCluster(tt.obj, tt.createns) //nolint:errcheck

		_, err := k2o.kudoClientset.
			KudoV1beta1().
			Instances(tt.createns).
			Get(context.TODO(), tt.name, metav1.GetOptions{})
		if tt.err != "" {
			assert.EqualError(t, err, tt.err, "failure in %v test case", i+1)
		}
	}
}

func TestKudoClient_GetInstance(t *testing.T) {
	testInstance := kudoapi.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				kudo.OperatorLabel: "test",
			},
			Name: "test",
		},
		Spec: kudoapi.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "test-1.0",
			},
		},
	}

	tests := []struct {
		name             string
		found            bool
		namespaceToQuery string
		storedInstance   *kudoapi.Instance
	}{
		{name: "no instance exists", namespaceToQuery: installNamespace},                                             // 1
		{name: "instance exists", found: true, namespaceToQuery: installNamespace, storedInstance: &testInstance},    // 2
		{name: "instance exists in different namespace", namespaceToQuery: "otherns", storedInstance: &testInstance}, // 3
	}

	for i, tt := range tests {
		k2o := newTestSimpleK2o()

		// create Instance
		if tt.storedInstance != nil {
			_, err := k2o.kudoClientset.
				KudoV1beta1().
				Instances(installNamespace).
				Create(context.TODO(), tt.storedInstance, metav1.CreateOptions{})
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
	testOv := kudoapi.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "OperatorVersion",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-1.0", operatorName),
		},
		Spec: kudoapi.OperatorVersionSpec{
			Version: "1.0",
		},
	}

	tests := []struct {
		name      string
		found     bool
		namespace string
		storedOv  *kudoapi.OperatorVersion
	}{
		{name: "no operator version defined", namespace: installNamespace},
		{name: "operator version exists in the same namespace", found: true, namespace: installNamespace, storedOv: &testOv},
		{name: "operator version exists in different namespace", namespace: "otherns", storedOv: &testOv},
	}

	for _, tt := range tests {
		k2o := newTestSimpleK2o()

		// create Instance
		if tt.storedOv != nil {
			_, err := k2o.kudoClientset.
				KudoV1beta1().
				OperatorVersions(installNamespace).
				Create(context.TODO(), tt.storedOv, metav1.CreateOptions{})
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
	testInstance := kudoapi.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				kudo.OperatorLabel: "test",
			},
			Name: "test",
		},
		Spec: kudoapi.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "test-1.0",
			},
		},
	}

	tests := []struct {
		name               string
		patchToVersion     *string
		existingParameters map[string]string
		parametersToPatch  map[string]string
		namespace          string
		triggeredPlan      *string
	}{
		{name: "patch to version", patchToVersion: convert.StringPtr("test-1.1.1"), namespace: installNamespace},
		{name: "patch adding new parameter", patchToVersion: convert.StringPtr("test-1.1.1"), parametersToPatch: map[string]string{"param": "value"}, namespace: installNamespace},
		{name: "patch updating parameter", patchToVersion: convert.StringPtr("test-1.1.1"), existingParameters: map[string]string{"param": "value"}, parametersToPatch: map[string]string{"param": "value2"}, namespace: installNamespace},
		{name: "do not patch the version", existingParameters: map[string]string{"param": "value"}, parametersToPatch: map[string]string{"param": "value2"}, namespace: installNamespace},
		{name: "patch with existing parameter should not override", patchToVersion: convert.StringPtr("1.1.1"), existingParameters: map[string]string{"param": "value"}, parametersToPatch: map[string]string{"other": "value2"}, namespace: installNamespace},
		{name: "patch with a new plan to execute", patchToVersion: convert.StringPtr("1.1.1"), existingParameters: map[string]string{"param": "value"}, namespace: installNamespace},
	}

	for _, tt := range tests {
		k2o := newTestSimpleK2o()

		// create Instance
		instanceToCreate := testInstance
		instanceToCreate.Spec.Parameters = tt.existingParameters
		_, err := k2o.kudoClientset.
			KudoV1beta1().
			Instances(installNamespace).
			Create(context.TODO(), &instanceToCreate, metav1.CreateOptions{})
		if err != nil {
			t.Errorf("Error creating operator version in tests setup for %s", tt.name)
		}

		err = k2o.UpdateInstance(testInstance.Name, installNamespace, tt.patchToVersion, tt.parametersToPatch, nil, false, 0)
		instance, _ := k2o.GetInstance(testInstance.Name, installNamespace)
		if tt.patchToVersion != nil {
			if err != nil || instance.Spec.OperatorVersion.Name != convert.StringValue(tt.patchToVersion) {
				t.Errorf("%s:\nexpected version: %v\n     got: %v, err: %v", tt.name, convert.StringValue(tt.patchToVersion), instance.Spec.OperatorVersion.Name, err)
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

		// triggered plan
		if tt.triggeredPlan != nil && err != nil {
			assert.Equal(t, *tt.triggeredPlan, instance.Spec.PlanExecution.PlanName)
			assert.NotEmpty(t, instance.Spec.PlanExecution.UID)

		}
	}
}

func TestKudoClient_DeleteInstance(t *testing.T) {
	testInstance := kudoapi.Instance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kudo.dev/v1beta1",
			Kind:       "Instance",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: kudoapi.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: "test-1.0",
			},
		},
	}

	tests := []struct {
		name         string
		instanceName string
		namespace    string
		shouldFail   bool
	}{
		{name: "non-existing instance", instanceName: "nonexisting-instance", namespace: installNamespace, shouldFail: true},
		{name: "non-existing namespace", instanceName: testInstance.Name, namespace: "otherns", shouldFail: true},
		{name: "delete instance", instanceName: testInstance.Name, namespace: installNamespace},
	}

	for _, test := range tests {
		k2o := newTestSimpleK2o()

		_, err := k2o.kudoClientset.
			KudoV1beta1().
			Instances(installNamespace).
			Create(context.TODO(), &testInstance, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("error creating instance in tests setup for")
		}

		err = k2o.DeleteInstance(test.instanceName, test.namespace)
		if err == nil {
			if test.shouldFail {
				t.Errorf("expected test %s to fail", test.name)
			} else {
				instance, err := k2o.GetInstance(test.instanceName, test.namespace)
				if err != nil {
					t.Errorf("failed to get instance: %v", err)
				}

				if instance != nil {
					t.Errorf("instance is still retrieved after being deleted in test %s", test.name)
				}
			}

		} else {
			if !test.shouldFail {
				t.Errorf("expected test %s to succeed but got error: %v", test.name, err)
			}
		}
	}
}

func TestKudoClient_CreateNamespace(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		manifest   string
		shouldFail bool
	}{
		{
			name:       "Invalid manifest",
			namespace:  "foo-test",
			manifest:   "invalid namespace resource",
			shouldFail: true,
		},
		{
			name:       "Namespace without manifest",
			namespace:  "foo-test",
			manifest:   "",
			shouldFail: false,
		},
		{
			name:      "Namespace name overwrites manifest",
			namespace: "foo-test",
			manifest: `apiVersion: v1
kind: Namespace
metadata:
  name: bar-test
`,
			shouldFail: false,
		},
	}

	for _, test := range tests {
		k2o := newTestSimpleK2o()

		err := k2o.CreateNamespace(test.namespace, test.manifest)
		if err == nil {
			if test.shouldFail {
				t.Errorf("expected test %s to fail", test.name)
			} else {
				namespace, err := k2o.KubeClientset.
					CoreV1().
					Namespaces().
					Get(context.TODO(), test.namespace, metav1.GetOptions{})
				assert.NoError(t, err)

				assert.Equal(t, namespace.Annotations["created-by"], "kudo-cli")
			}
		} else {
			if !test.shouldFail {
				t.Errorf("expected test %s to succeed but got error: %v", test.name, err)
			}
		}
	}
}
