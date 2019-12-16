package prereq

import (
	"testing"

	"github.com/stretchr/testify/assert"
	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	apiextensionfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	testing2 "k8s.io/client-go/testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verify"
)

func getFakeClient() *kube.Client {
	client := fake.NewSimpleClientset()

	extClient := apiextensionfake.NewSimpleClientset()

	dynamicClient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())

	return &kube.Client{
		KubeClient:    client,
		ExtClient:     extClient,
		DynamicClient: dynamicClient}
}

func TestPrereq_Ok_PreValidate_DefaultOpts(t *testing.T) {
	client := getFakeClient()

	init := NewInitializer(kudoinit.NewOptions("", "", "", make([]string, 0)))

	result := init.PreInstallVerify(client)

	assert.EqualValues(t, verify.NewResult(), result)
}

func mockGetNamespace(client *kube.Client, nsName string) {
	client.KubeClient.(*fake.Clientset).Fake.PrependReactor("get", "namespaces", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
		ns := &core.Namespace{
			ObjectMeta: v12.ObjectMeta{
				Name: nsName,
			},
		}
		return true, ns, nil
	})
}

func mockListServiceAccounts(client *kube.Client, saName string) {
	client.KubeClient.(*fake.Clientset).Fake.PrependReactor("list", "serviceaccounts", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
		sa := core.ServiceAccount{
			ObjectMeta: v12.ObjectMeta{
				Name: saName,
			},
			Secrets:                      nil,
			ImagePullSecrets:             nil,
			AutomountServiceAccountToken: nil,
		}
		sal := &core.ServiceAccountList{
			Items: []core.ServiceAccount{sa},
		}
		return true, sal, nil
	})
}

func mockListClusterRoleBindings(client *kube.Client, opts kudoinit.Options) {
	client.KubeClient.(*fake.Clientset).Fake.PrependReactor("list", "clusterrolebindings", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
		subject := rbac.Subject{
			Kind:      "",
			Name:      opts.ServiceAccount,
			Namespace: opts.Namespace,
		}

		crb := rbac.ClusterRoleBinding{
			Subjects: []rbac.Subject{subject},
			RoleRef: rbac.RoleRef{
				Name: "cluster-admin",
			},
		}
		crbList := &rbac.ClusterRoleBindingList{
			Items: []rbac.ClusterRoleBinding{crb},
		}
		return true, crbList, nil
	})
}
