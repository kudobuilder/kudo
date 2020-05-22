package diagnostics

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	kubefake "k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

const (
	fakeNamespace  = "my-namespace"
	fakeZkInstance = "zookeeper-instance"
)

const (
	zkOperatorFile        = "diag/operator_zookeeper/zookeeper.yaml"
	zkOperatorVersionFile = "diag/operator_zookeeper/operatorversion_zookeeper-0.3.0/zookeeper-0.3.0.yaml"
	zkPod2File            = "diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-2/zookeeper-instance-zookeeper-2.yaml"
	zkLog2File            = "diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-2/zookeeper-instance-zookeeper-2.log.gz"
	zkServicesFile        = "diag/operator_zookeeper/instance_zookeeper-instance/servicelist.yaml"
	zkPod0File            = "diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-0/zookeeper-instance-zookeeper-0.yaml"
	zkLog0File            = "diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-0/zookeeper-instance-zookeeper-0.log.gz"
	zkInstanceFile        = "diag/operator_zookeeper/instance_zookeeper-instance/zookeeper-instance.yaml"
	zkPod1File            = "diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-1/zookeeper-instance-zookeeper-1.yaml"
	zkLog1File            = "diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-1/zookeeper-instance-zookeeper-1.log.gz"
	zkStatefulSetsFile    = "diag/operator_zookeeper/instance_zookeeper-instance/statefulsetlist.yaml"
	versionFile           = "diag/version.yaml"
	kmServicesFile        = "diag/kudo/servicelist.yaml"
	kmPodFile             = "diag/kudo/pod_kudo-controller-manager-0/kudo-controller-manager-0.yaml"
	kmLogFile             = "diag/kudo/pod_kudo-controller-manager-0/kudo-controller-manager-0.log.gz"
	kmServiceAccountsFile = "diag/kudo/serviceaccountlist.yaml"
	kmStatefulSetsFile    = "diag/kudo/statefulsetlist.yaml"
	settingsFile          = "diag/settings.yaml"
)

// defaultFileNames - all the files that should be created if no error happens
func defaultFileNames() map[string]struct{} {
	return map[string]struct{}{
		zkOperatorFile:        {},
		zkOperatorVersionFile: {},
		zkPod2File:            {},
		zkLog2File:            {},
		zkServicesFile:        {},
		zkPod0File:            {},
		zkLog0File:            {},
		zkInstanceFile:        {},
		zkPod1File:            {},
		zkLog1File:            {},
		zkStatefulSetsFile:    {},
		versionFile:           {},
		kmServicesFile:        {},
		kmPodFile:             {},
		kmLogFile:             {},
		kmServiceAccountsFile: {},
		kmStatefulSetsFile:    {},
		settingsFile:          {},
	}
}

var errFakeTestError = fmt.Errorf("fake test error")

// resource to be loaded into fake clients
var (
	// resource of the instance for which diagnostics is run
	pods            corev1.PodList
	serviceAccounts corev1.ServiceAccountList
	services        corev1.ServiceList
	statefulsets    appsv1.StatefulSetList
	pvs             corev1.PersistentVolumeList
	pvcs            corev1.PersistentVolumeClaimList
	operator        v1beta1.Operator
	operatorVersion v1beta1.OperatorVersion
	instance        v1beta1.Instance

	// kudo-manager resources
	kmNs              corev1.Namespace
	kmPod             corev1.Pod
	kmServices        corev1.ServiceList
	kmServiceAccounts corev1.ServiceAccountList
	kmStatefulsets    appsv1.StatefulSetList

	// resources unrelated to the diagnosed instance or kudo-manager, should not be collected
	cowPod                corev1.Pod
	defaultServiceAccount corev1.ServiceAccount
	clusterRole           rbacv1beta1.ClusterRole
)

var (
	kubeObjects objectList
	kudoObjects objectList
)

func check(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func assertNilError(t *testing.T) func(error) {
	return func(e error) {
		assert.Nil(t, e)
	}
}

func mustReadObjectFromYaml(fs afero.Fs, fname string, object runtime.Object, check func(error)) {
	b, err := afero.ReadFile(fs, fname)
	check(err)
	j, err := yaml.YAMLToJSON(b)
	check(err)
	err = json.Unmarshal(j, object)
	check(err)
}

type objectList []runtime.Object

func (l objectList) append(o runtime.Object) objectList {
	if meta.IsListType(o) {
		objs, err := meta.ExtractList(o)
		check(err)
		return append(l, objs...)
	}
	return append(l, o)
}

func init() {
	osFs := afero.NewOsFs()
	mustReadObjectFromYaml(osFs, "testdata/zk_pods.yaml", &pods, check)
	mustReadObjectFromYaml(osFs, "testdata/zk_service_accounts.yaml", &serviceAccounts, check)
	mustReadObjectFromYaml(osFs, "testdata/zk_services.yaml", &services, check)
	mustReadObjectFromYaml(osFs, "testdata/zk_statefulsets.yaml", &statefulsets, check)
	mustReadObjectFromYaml(osFs, "testdata/zk_pvs.yaml", &pvs, check)
	mustReadObjectFromYaml(osFs, "testdata/zk_pvcs.yaml", &pvcs, check)
	mustReadObjectFromYaml(osFs, "testdata/zk_operator.yaml", &operator, check)
	mustReadObjectFromYaml(osFs, "testdata/zk_operatorversion.yaml", &operatorVersion, check)
	mustReadObjectFromYaml(osFs, "testdata/zk_instance.yaml", &instance, check)
	mustReadObjectFromYaml(osFs, "testdata/kudo_ns.yaml", &kmNs, check)
	mustReadObjectFromYaml(osFs, "testdata/kudo_pod.yaml", &kmPod, check)
	mustReadObjectFromYaml(osFs, "testdata/kudo_services.yaml", &kmServices, check)
	mustReadObjectFromYaml(osFs, "testdata/kudo_serviceaccounts.yaml", &kmServiceAccounts, check)
	mustReadObjectFromYaml(osFs, "testdata/kudo_statefulsets.yaml", &kmStatefulsets, check)
	mustReadObjectFromYaml(osFs, "testdata/cow_pod.yaml", &cowPod, check)
	mustReadObjectFromYaml(osFs, "testdata/kudo_default_serviceaccount.yaml", &defaultServiceAccount, check)
	mustReadObjectFromYaml(osFs, "testdata/cluster_role.yaml", &clusterRole, check)

	// standard kube objects to be returned by kube clientset
	kubeObjects = objectList{}.
		append(&pods).
		append(&serviceAccounts).
		append(&services).
		append(&statefulsets).
		append(&pvs).
		append(&pvcs).
		append(&kmNs).
		append(&kmPod).
		append(&kmServices).
		append(&kmServiceAccounts).
		append(&kmStatefulsets).
		append(&cowPod).
		append(&defaultServiceAccount).
		append(&clusterRole)

	// kudo custom resources to be returned by kudo clientset
	kudoObjects = objectList{}.
		append(&operator).
		append(&operatorVersion).
		append(&instance)
}

func TestCollect_OK(t *testing.T) {
	k8cs := kubefake.NewSimpleClientset(kubeObjects...)
	kcs := fake.NewSimpleClientset(kudoObjects...)
	client := kudo.NewClientFromK8s(kcs, k8cs)

	fs := &afero.MemMapFs{}
	err := Collect(fs, &Options{
		Instance: fakeZkInstance,
		LogSince: -1,
	}, client, &env.Settings{
		Namespace: fakeNamespace,
	})
	assert.Nil(t, err)

	// all files should be present and no other files
	fileNames := defaultFileNames()
	for name := range fileNames {
		exists, _ := afero.Exists(fs, name)
		assert.True(t, exists, "file %s not found", name)
	}
	_ = afero.Walk(fs, "diag", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			_, ok := fileNames[path]
			assert.True(t, ok, "unexpected file: %s", path)
		}
		return nil
	})

	var (
		collectedPod0              corev1.Pod
		collectedPod1              corev1.Pod
		collectedPod2              corev1.Pod
		collectedKmPod             corev1.Pod
		collectedServices          corev1.ServiceList
		collectedStatefulsets      appsv1.StatefulSetList
		collectedOperator          v1beta1.Operator
		collectedOperatorVersion   v1beta1.OperatorVersion
		collectedInstance          v1beta1.Instance
		collectedKmServices        corev1.ServiceList
		collectedKmServiceAccounts corev1.ServiceAccountList
		collectedKmStatefulsets    appsv1.StatefulSetList
	)

	// read the created files and assert no error
	mustReadObjectFromYaml(fs, zkOperatorFile, &collectedOperator, assertNilError(t))
	mustReadObjectFromYaml(fs, zkOperatorVersionFile, &collectedOperatorVersion, assertNilError(t))
	mustReadObjectFromYaml(fs, zkPod2File, &collectedPod2, assertNilError(t))
	mustReadObjectFromYaml(fs, zkServicesFile, &collectedServices, assertNilError(t))
	mustReadObjectFromYaml(fs, zkPod0File, &collectedPod0, assertNilError(t))
	mustReadObjectFromYaml(fs, zkInstanceFile, &collectedInstance, assertNilError(t))
	mustReadObjectFromYaml(fs, zkPod1File, &collectedPod1, assertNilError(t))
	mustReadObjectFromYaml(fs, zkStatefulSetsFile, &collectedStatefulsets, assertNilError(t))
	mustReadObjectFromYaml(fs, kmServicesFile, &collectedKmServices, assertNilError(t))
	mustReadObjectFromYaml(fs, kmPodFile, &collectedKmPod, assertNilError(t))
	mustReadObjectFromYaml(fs, kmServiceAccountsFile, &collectedKmServiceAccounts, assertNilError(t))
	mustReadObjectFromYaml(fs, kmStatefulSetsFile, &collectedKmStatefulsets, assertNilError(t))

	// verify the correctness of the created files by comparison of the objects read from those to the original objects
	assert.Equal(t, operator, collectedOperator)
	assert.Equal(t, operatorVersion, collectedOperatorVersion)
	assert.Equal(t, pods.Items[2], collectedPod2)
	assert.Equal(t, services, collectedServices)
	assert.Equal(t, pods.Items[0], collectedPod0)
	assert.Equal(t, instance, collectedInstance)
	assert.Equal(t, pods.Items[1], collectedPod1)
	assert.Equal(t, statefulsets, collectedStatefulsets)
	assert.Equal(t, kmServices, collectedKmServices)
	assert.Equal(t, kmPod, collectedKmPod)
	assert.Equal(t, kmServiceAccounts, collectedKmServiceAccounts)
	assert.Equal(t, kmStatefulsets, collectedKmStatefulsets)
}

// Fatal error
func TestCollect_FatalError(t *testing.T) {
	k8cs := kubefake.NewSimpleClientset(kubeObjects...)
	kcs := fake.NewSimpleClientset(kudoObjects...)

	// force kudo clientset to return no Operator
	reactor := func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
		if action.GetNamespace() == fakeNamespace {
			return true, nil, nil
		}
		return
	}
	kcs.PrependReactor("get", "operators", reactor)

	client := kudo.NewClientFromK8s(kcs, k8cs)
	fs := &afero.MemMapFs{}
	err := Collect(fs, &Options{
		Instance: fakeZkInstance,
		LogSince: -1,
	}, client, &env.Settings{
		Namespace: fakeNamespace,
	})

	assert.NotNil(t, err)
}

// Fatal error - special case: api server returns "Not Found", api then returns (nil, nil)
func TestCollect_FatalNotFound(t *testing.T) {
	k8cs := kubefake.NewSimpleClientset(kubeObjects...)
	kcs := fake.NewSimpleClientset(kudoObjects...)

	// force kudo clientset to return no Operator
	reactor := func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
		if action.GetNamespace() == fakeNamespace {
			err := errors.NewNotFound(schema.GroupResource{
				Group:    "kudo.dev/v1beta1",
				Resource: "operators",
			}, "zookeeper")
			return true, nil, err
		}
		return
	}
	kcs.PrependReactor("get", "operators", reactor)

	client := kudo.NewClientFromK8s(kcs, k8cs)
	fs := &afero.MemMapFs{}
	err := Collect(fs, &Options{
		Instance: fakeZkInstance,
		LogSince: -1,
	}, client, &env.Settings{
		Namespace: fakeNamespace,
	})

	assert.NotNil(t, err)
}

// Client returns an error retrieving a resource that should not be wrapped into its own dir
// corresponding resource collector has  failOnError = false
func TestCollect_NonFatalError(t *testing.T) {
	k8cs := kubefake.NewSimpleClientset(kubeObjects...)
	kcs := fake.NewSimpleClientset(kudoObjects...)

	// force kube clientset to return error when retrieving services
	reactor := func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
		if action.GetNamespace() == fakeNamespace {
			return true, nil, errFakeTestError
		}
		return
	}
	k8cs.PrependReactor("list", "services", reactor)

	client := kudo.NewClientFromK8s(kcs, k8cs)
	fs := &afero.MemMapFs{}
	err := Collect(fs, &Options{
		Instance: fakeZkInstance,
		LogSince: -1,
	}, client, &env.Settings{
		Namespace: fakeNamespace,
	})

	// no error returned, error is saved into the file in place of the corresponding resource file
	assert.Nil(t, err)
	exists, _ := afero.Exists(fs, zkServicesFile)
	assert.False(t, exists)
	exists, _ = afero.Exists(fs, "diag/operator_zookeeper/instance_zookeeper-instance/service.err")
	assert.True(t, exists)
}

// Client returns an error retrieving a resource to be printed in its own dir
// corresponding resource collector has  failOnError = false
func TestCollect_NonFatalErrorWithDir(t *testing.T) {
	k8cs := kubefake.NewSimpleClientset(kubeObjects...)
	kcs := fake.NewSimpleClientset(kudoObjects...)

	// force kube clientset to return error when retrieving pods
	reactor := func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
		if action.GetNamespace() == fakeNamespace {
			return true, nil, errFakeTestError
		}
		return
	}
	k8cs.PrependReactor("list", "pods", reactor)
	client := kudo.NewClientFromK8s(kcs, k8cs)
	fs := &afero.MemMapFs{}
	err := Collect(fs, &Options{
		Instance: fakeZkInstance,
		LogSince: -1,
	}, client, &env.Settings{
		Namespace: fakeNamespace,
	})

	// no error returned, no pods files present, error file present in the directory where otherwise pod dirs would have been
	assert.Nil(t, err)
	exists, _ := afero.Exists(fs, zkPod2File)
	assert.False(t, exists)
	exists, _ = afero.Exists(fs, zkPod0File)
	assert.False(t, exists)
	exists, _ = afero.Exists(fs, zkPod1File)
	assert.False(t, exists)
	exists, _ = afero.Exists(fs, "diag/operator_zookeeper/instance_zookeeper-instance/pod.err")
	assert.True(t, exists)
}

func TestCollect_KudoNameSpaceNotFound(t *testing.T) {
	k8cs := kubefake.NewSimpleClientset(kubeObjects...)
	kcs := fake.NewSimpleClientset(kudoObjects...)

	// force kudo clientset to return no Operator
	reactor := func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
		err = errors.NewNotFound(schema.GroupResource{
			Group:    "kudo.dev/v1beta1",
			Resource: "namespaces",
		}, "kudo-system")
		return true, nil, err
	}
	k8cs.PrependReactor("list", "namespaces", reactor)

	client := kudo.NewClientFromK8s(kcs, k8cs)
	fs := &afero.MemMapFs{}
	err := Collect(fs, &Options{
		Instance: fakeZkInstance,
		LogSince: -1,
	}, client, &env.Settings{
		Namespace: fakeNamespace,
	})

	assert.NotNil(t, err)
}

// failingFs is a wrapper of afero.Fs to simulate a specific file creation failure for printer
type failingFs struct {
	afero.Fs
	failOn string
}

func (s *failingFs) Create(name string) (afero.File, error) {
	if name == s.failOn {
		return nil, errFakeTestError
	}
	return s.Fs.Create(name)
}

func TestCollect_PrintFailure(t *testing.T) {
	k8cs := kubefake.NewSimpleClientset(kubeObjects...)
	kcs := fake.NewSimpleClientset(kudoObjects...)
	client := kudo.NewClientFromK8s(kcs, k8cs)

	a := &afero.MemMapFs{}
	fs := &failingFs{Fs: a, failOn: zkPod2File}

	err := Collect(fs, &Options{
		Instance: fakeZkInstance,
		LogSince: -1,
	}, client, &env.Settings{
		Namespace: fakeNamespace,
	})
	assert.NotNil(t, err)

	// all files should be present except the one failed to be printed, and no other files
	fileNames := defaultFileNames()
	delete(fileNames, zkPod2File)

	for name := range fileNames {
		exists, _ := afero.Exists(fs, name)
		assert.True(t, exists, "file %s not found", name)
	}
	_ = afero.Walk(fs, "diag", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			_, ok := fileNames[path]
			assert.True(t, ok, "unexpected file: %s", path)
		}
		return nil
	})
}
