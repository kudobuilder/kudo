package diagnostics

import (
	"github.com/ghodss/yaml"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/client/clientset/versioned/fake"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	//"github.com/spf13/afero"
	"io/ioutil"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/json"
	"log"
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"

	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func check(err error) {
	if err !=nil {
		log.Fatalln(err)
	}
}

func mustReadObjectFromYaml(fname string, object runtime.Object) {
	b, err := ioutil.ReadFile(fname)
	check(err)
	j, err :=  yaml.YAMLToJSON(b)
	check(err)
	err = json.Unmarshal(j, object)
	check(err)
}

type objectList []runtime.Object

func (l objectList) append(o runtime.Object) objectList{
	if meta.IsListType(o) {
		objs, err := meta.ExtractList(o)
		check(err)
		return append(l, objs...)
	}
	return append(l, o)
}

type testClientFactory struct {
	cs *kubefake.Clientset
	ks *fake.Clientset
}

func (f *testClientFactory) KubeClient() (*kube.Client, error) {
	return &kube.Client{
		KubeClient:    f.cs,
		ExtClient:     nil,
		DynamicClient: nil,
	}, nil
}

func (f *testClientFactory) KudoClient() (*kudo.Client, error) {
	return kudo.NewClientFromK8s(f.ks, f.cs), nil
}

func TestCollect(t *testing.T) {
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

	kubeObjects := objectList{}.
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

	kudoObjects := objectList{}.
		append(&operator).
		append(&operatorVersion).
		append(&instance)

	c := kudo.NewClientFromK8s(fake.NewSimpleClientset(kudoObjects...), kubefake.NewSimpleClientset(kubeObjects...))
	fs := &afero.MemMapFs{}
	err := Collect(fs, &Options{
		Instance: "zookeeper-instance",
		LogSince: -1,
	}, c, &env.Settings{
		KubeConfig:     "",
		Home:           "",
		Namespace:      "my-namespace",
		RequestTimeout: 0,
		Validate:       false,
	})
	check(err)
	
	fileNames := []string{
		"diag/operator_zookeeper/zookeeper.yaml",
		"diag/operator_zookeeper/operatorversion_zookeeper-0.3.0",
		"diag/operator_zookeeper/operatorversion_zookeeper-0.3.0/zookeeper-0.3.0.yaml",
		"diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-2/zookeeper-instance-zookeeper-2.yaml",
		"diag/operator_zookeeper/instance_zookeeper-instance/servicelist.yaml",
		"diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-0/zookeeper-instance-zookeeper-0.yaml",
		"diag/operator_zookeeper/instance_zookeeper-instance/zookeeper-instance.yaml",
		"diag/operator_zookeeper/instance_zookeeper-instance/pod_zookeeper-instance-zookeeper-1/zookeeper-instance-zookeeper-1.yaml",
		"diag/operator_zookeeper/instance_zookeeper-instance/statefulsetlist.yaml",
		"diag/version.yaml",
		"diag/kudo/servicelist.yaml",
		"diag/kudo/pod_kudo-controller-manager-0/kudo-controller-manager-0.yaml",
		"diag/kudo/pod_kudo-controller-manager-0/kudo-controller-manager-0.log.gz",
		"diag/kudo/serviceaccountlist.yaml",
		"diag/kudo/statefulsetlist.yaml",
		"diag/settings.yaml",
	}
	for _, name := range fileNames {
		exists,_ := afero.Exists(fs, name)
		assert.True(t, exists, "file %s not found", name)
	}
}
