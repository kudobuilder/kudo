package prereq

import (
	"fmt"
	"reflect"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

// Ensure IF is implemented
var _ k8sResource = &kudoServiceAccount{}

type kudoServiceAccount struct {
	opts           kudoinit.Options
	serviceAccount *v1.ServiceAccount
	roleBinding    *rbacv1.ClusterRoleBinding
}

func newServiceAccount(options kudoinit.Options) kudoServiceAccount {
	return kudoServiceAccount{
		opts:           options,
		serviceAccount: generateServiceAccount(options),
		roleBinding:    generateRoleBinding(options),
	}
}

func (o kudoServiceAccount) PreInstallVerify(client *kube.Client) verifier.Result {
	result := verifier.NewResult()
	if o.opts.IsDefaultServiceAccount() {
		return result
	}
	result.Merge(o.validateServiceAccountExists(client))

	if result.IsValid() {
		// Only validate role if SA is ok
		result.Merge(o.validateClusterAdminRoleForSA(client))
	}

	return result
}

func (o kudoServiceAccount) Install(client *kube.Client) error {
	if !o.opts.IsDefaultServiceAccount() {
		return nil
	}
	if err := o.installServiceAccount(client); err != nil {
		return err
	}
	if err := o.installRoleBinding(client); err != nil {
		return err
	}
	return nil
}

// Validate whether the serviceAccount exists
func (o kudoServiceAccount) validateServiceAccountExists(client *kube.Client) verifier.Result {
	coreClient := client.KubeClient.CoreV1()
	saList, err := coreClient.ServiceAccounts(o.opts.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return verifier.NewError(fmt.Sprintf("Failed to validate that service account %s exists: %v", o.opts.ServiceAccount, err))
	}
	for _, sa := range saList.Items {
		if sa.Name == o.opts.ServiceAccount {
			return verifier.NewResult()
		}
	}
	return verifier.NewError(fmt.Sprintf("Service Account %s does not exists - KUDO expects the serviceAccount to be present in the namespace %s", o.opts.ServiceAccount, o.opts.Namespace))
}

// Validate whether the serviceAccount has cluster-admin role
func (o kudoServiceAccount) validateClusterAdminRoleForSA(client *kube.Client) verifier.Result {
	// Check whether the serviceAccount has clusterrolebinding cluster-admin
	crbs, err := client.KubeClient.RbacV1().ClusterRoleBindings().List(metav1.ListOptions{})
	if err != nil {
		return verifier.NewError(fmt.Sprintf("Failed to validate role bindings: %v", err))
	}

	for _, crb := range crbs.Items {
		for _, subject := range crb.Subjects {
			if subject.Name == o.opts.ServiceAccount && subject.Namespace == o.opts.Namespace && crb.RoleRef.Name == "cluster-admin" {
				return verifier.NewResult()
			}
		}
	}

	return verifier.NewError(fmt.Sprintf("Service Account %s does not have cluster-admin role - KUDO expects the serviceAccount passed to be in the namespace %s and to have cluster-admin role", o.opts.ServiceAccount, o.opts.Namespace))
}

func (o kudoServiceAccount) installServiceAccount(client *kube.Client) error {
	coreClient := client.KubeClient.CoreV1()
	_, err := coreClient.ServiceAccounts(o.opts.Namespace).Create(o.serviceAccount)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("service account %v already exists", o.serviceAccount.Name)
		return nil
	}
	return err
}

func (o kudoServiceAccount) installRoleBinding(client *kube.Client) error {
	_, err := client.KubeClient.RbacV1().ClusterRoleBindings().Create(o.roleBinding)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("role binding %v already exists", o.roleBinding.Name)
		return nil
	}
	return err
}

func (o kudoServiceAccount) ValidateInstallation(client *kube.Client) error {
	coreClient := client.KubeClient.CoreV1()

	existingSA, err := coreClient.ServiceAccounts(o.opts.Namespace).Get(o.serviceAccount.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to retrieve service account %v", err)
	}

	if !reflect.DeepEqual(existingSA, o.serviceAccount) {
		return fmt.Errorf("installed ServiceAccount does not equal expected service account")
	}

	existingRB, err := client.KubeClient.RbacV1().RoleBindings(o.opts.Namespace).Get(o.roleBinding.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to retrieve role binding %v", err)
	}

	if !reflect.DeepEqual(existingRB, o.roleBinding) {
		return fmt.Errorf("installed ClusterRoleBinding does not equal expected")
	}
	return nil
}

func (o kudoServiceAccount) AsRuntimeObjs() []runtime.Object {
	if o.opts.IsDefaultServiceAccount() {
		return []runtime.Object{o.serviceAccount, o.roleBinding}
	}
	return make([]runtime.Object, 0)
}

// generateServiceAccount builds the system account
func generateServiceAccount(opts kudoinit.Options) *v1.ServiceAccount {
	labels := kudoinit.GenerateLabels(map[string]string{})
	return &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      opts.ServiceAccount,
			Namespace: opts.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
	}
}

func generateRoleBinding(opts kudoinit.Options) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kudo-manager-rolebinding",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      opts.ServiceAccount,
			Namespace: opts.Namespace,
		}},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
	}
}
