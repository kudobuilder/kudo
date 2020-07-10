package prereq

import (
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/assert"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	testing2 "k8s.io/client-go/testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

func TestPrereq_Fail_PreValidate_CustomServiceAccount(t *testing.T) {
	client := getFakeClient()

	init := NewServiceAccountInitializer(kudoinit.NewOptions("", "", "customSA", false, true))

	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.EqualValues(t, verifier.NewError("Service Account customSA does not exists - KUDO expects the serviceAccount to be present in the namespace kudo-system"), result)
}

func TestPrereq_Fail_PreValidate_CustomServiceAccount_MissingPermissions(t *testing.T) {
	client := getFakeClient()

	customSA := "customSA"

	mockListServiceAccounts(client, customSA)

	init := NewServiceAccountInitializer(kudoinit.NewOptions("", "", "customSA", false, true))

	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.EqualValues(t, verifier.NewError("Service Account customSA does not have cluster-admin role - KUDO expects the serviceAccount passed to be in the namespace kudo-system and to have cluster-admin role"), result)
}

func TestPrereq_Ok_PreValidate_CustomServiceAccount(t *testing.T) {
	client := getFakeClient()

	customSA := "customSA"
	opts := kudoinit.NewOptions("", "", customSA, false, true)

	mockListServiceAccounts(client, opts.ServiceAccount)
	mockListClusterRoleBindings(client, opts)

	init := NewServiceAccountInitializer(opts)
	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.EqualValues(t, verifier.NewResult(), result)
}

func mockListServiceAccounts(client *kube.Client, saName string) {
	client.KubeClient.(*fake.Clientset).Fake.PrependReactor("list", "serviceaccounts", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
		sa := v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: saName,
			},
			Secrets:                      nil,
			ImagePullSecrets:             nil,
			AutomountServiceAccountToken: nil,
		}
		sal := &v1.ServiceAccountList{
			Items: []v1.ServiceAccount{sa},
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
