package diagnostics

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
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

var errFakeTestError = fmt.Errorf("fake test error")

var (
	kubeObjects objectList
	kudoObjects objectList
)

func check(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func mustReadObjectFromYaml(fname string, object runtime.Object) {
	b, err := ioutil.ReadFile(fname)
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
	var ns v12.Namespace
	mustReadObjectFromYaml("testdata/zk_namespace.yaml", &ns)
	var pods v12.PodList
	mustReadObjectFromYaml("testdata/zk_pods.yaml", &pods)
	var serviceAccounts v12.ServiceAccountList
	mustReadObjectFromYaml("testdata/zk_service_accounts.yaml", &serviceAccounts)
	var services v12.ServiceList
	mustReadObjectFromYaml("testdata/zk_services.yaml", &services)
	var statefulset v1.StatefulSet
	mustReadObjectFromYaml("testdata/zk_statefulset.yaml", &statefulset)
	var pvs v12.PersistentVolumeList
	mustReadObjectFromYaml("testdata/zk_pvs.yaml", &pvs)
	var pvcs v12.PersistentVolumeClaimList
	mustReadObjectFromYaml("testdata/zk_pvcs.yaml", &pvcs)
	var operator v1beta1.Operator
	mustReadObjectFromYaml("testdata/zk_operator.yaml", &operator)
	var operatorVersion v1beta1.OperatorVersion
	mustReadObjectFromYaml("testdata/zk_operatorversion.yaml", &operatorVersion)
	var instance v1beta1.Instance
	mustReadObjectFromYaml("testdata/zk_instance.yaml", &instance)
	var kmNs v12.Namespace
	mustReadObjectFromYaml("testdata/kudo_ns.yaml", &kmNs)
	var kmPod v12.Pod
	mustReadObjectFromYaml("testdata/kudo_pod.yaml", &kmPod)
	var kmService v12.Service
	mustReadObjectFromYaml("testdata/kudo_pod.yaml", &kmService)
	var kmServiceAccount v12.ServiceAccountList
	mustReadObjectFromYaml("testdata/kudo_serviceaccounts.yaml", &kmServiceAccount)
	var kmStatefulset v1.StatefulSet
	mustReadObjectFromYaml("testdata/kudo_pod.yaml", &kmStatefulset)

	// standard kube objects to be returned by kube clientset
	kubeObjects = objectList{}.
		append(&pods).
		append(&serviceAccounts).
		append(&services).
		append(&statefulset).
		append(&pvs).
		append(&pvcs).
		append(&kmNs).
		append(&kmPod).
		append(&kmService).
		append(&kmServiceAccount).
		append(&kmStatefulset)

	// kudo custom resources to be returned by kudo clientset
	kudoObjects = objectList{}.
		append(&operator).
		append(&operatorVersion).
		append(&instance)
}

// defaultFileNames - all the files that should be created if no error happens
func defaultFileNames() map[string]struct{} {
	return map[string]struct{}{
		"diag/operator_zookeeper/zookeeper.yaml":                                                                                       {},
		"diag/operator_zookeeper/operatorversion_zookeeper-0.3.0":                                                                      {},
		"diag/operator_zookeeper/operatorversion_zookeeper-0.3.0/zookeeper-0.3.0.yaml":                                                 {},
		"diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-2/zookeeper-instance-zookeeper-2.yaml":   {},
		"diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-2/zookeeper-instance-zookeeper-2.log.gz": {},
		"diag/operator_zookeeper/instance_zookeeper-instance/servicelist.yaml":                                                         {},
		"diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-0/zookeeper-instance-zookeeper-0.yaml":   {},
		"diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-0/zookeeper-instance-zookeeper-0.log.gz": {},
		"diag/operator_zookeeper/instance_zookeeper-instance/zookeeper-instance.yaml":                                                  {},
		"diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-1/zookeeper-instance-zookeeper-1.yaml":   {},
		"diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-1/zookeeper-instance-zookeeper-1.log.gz": {},
		"diag/operator_zookeeper/instance_zookeeper-instance/statefulsetlist.yaml":                                                     {},
		"diag/version.yaml":          {},
		"diag/kudo/servicelist.yaml": {},
		"diag/kudo/pod_kudo-controller-manager-0/kudo-controller-manager-0.yaml":   {},
		"diag/kudo/pod_kudo-controller-manager-0/kudo-controller-manager-0.log.gz": {},
		"diag/kudo/serviceaccountlist.yaml":                                        {},
		"diag/kudo/statefulsetlist.yaml":                                           {},
		"diag/settings.yaml":                                                       {},
	}
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
}

// Fatal error
func TestCollect_OperatorNotFound(t *testing.T) {
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

// Client returns an error retrieving a resource that should not be wrapped into its own dir
// corresponding resource collector has  failOnError = false
func TestCollect_NonFatal(t *testing.T) {
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
	exists, _ := afero.Exists(fs, "diag/operator_zookeeper/instance_zookeeper-instance/servicelist.yaml")
	assert.False(t, exists)
	exists, _ = afero.Exists(fs, "diag/operator_zookeeper/instance_zookeeper-instance/service.err")
	assert.True(t, exists)
}

// Client returns an error retrieving a resource to be printed in its own dir
// corresponding resource collector has  failOnError = false
func TestCollect_NonFatalWithDir(t *testing.T) {
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
	exists, _ := afero.Exists(fs, "diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-2/zookeeper-instance-zookeeper-2.yaml")
	assert.False(t, exists)
	exists, _ = afero.Exists(fs, "diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-0/zookeeper-instance-zookeeper-0.yaml")
	assert.False(t, exists)
	exists, _ = afero.Exists(fs, "diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-1/zookeeper-instance-zookeeper-1.yaml")
	assert.False(t, exists)
	exists, _ = afero.Exists(fs, "diag/operator_zookeeper/instance_zookeeper-instance/pod.err")
	assert.True(t, exists)
}

// failingFs is a wrapper of afero.Fs to simulate a specific file creation failure
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
	fs := &failingFs{Fs: a, failOn: "diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-2/zookeeper-instance-zookeeper-2.yaml"}

	err := Collect(fs, &Options{
		Instance: fakeZkInstance,
		LogSince: -1,
	}, client, &env.Settings{
		Namespace: fakeNamespace,
	})
	assert.NotNil(t, err)

	fileNames := defaultFileNames()
	delete(fileNames, "diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-2/zookeeper-instance-zookeeper-2.yaml")

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
