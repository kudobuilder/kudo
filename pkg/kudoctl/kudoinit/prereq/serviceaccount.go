package prereq

import (
	"context"
	"fmt"

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
var _ kudoinit.Step = &KudoServiceAccount{}

type KudoServiceAccount struct {
	opts           kudoinit.Options
	serviceAccount *v1.ServiceAccount
	roleBinding    *rbacv1.ClusterRoleBinding
}

func NewServiceAccountInitializer(options kudoinit.Options) KudoServiceAccount {
	return KudoServiceAccount{
		opts:           options,
		serviceAccount: generateServiceAccount(options),
		roleBinding:    generateRoleBinding(options),
	}
}

func (o KudoServiceAccount) String() string {
	return "service account"
}

func (o KudoServiceAccount) PreInstallVerify(client *kube.Client, result *verifier.Result) error {
	if o.opts.IsDefaultServiceAccount() {
		return nil
	}
	if err := o.validateServiceAccountExists(client, result); err != nil {
		return err
	}

	if result.IsValid() {
		// Only validate role if SA is ok
		if err := o.validateClusterAdminRoleForSA(client, result); err != nil {
			return err
		}
	}

	return nil
}

func (o KudoServiceAccount) PreUpgradeVerify(client *kube.Client, result *verifier.Result) error {
	// For the service account we just verify that the installation is valid. Nothing really to upgrade here
	return o.VerifyInstallation(client, result)
}

func (o KudoServiceAccount) VerifyInstallation(client *kube.Client, result *verifier.Result) error {
	if err := o.validateServiceAccountExists(client, result); err != nil {
		return err
	}

	if result.IsValid() {
		// Only validate role if SA is ok
		if err := o.validateClusterAdminRoleForSA(client, result); err != nil {
			return err
		}
	}

	return nil
}

func (o KudoServiceAccount) Install(client *kube.Client) error {
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

func (o KudoServiceAccount) Resources() []runtime.Object {
	if o.opts.IsDefaultServiceAccount() {
		return []runtime.Object{o.serviceAccount, o.roleBinding}
	}
	return make([]runtime.Object, 0)
}

// Validate whether the serviceAccount exists
func (o KudoServiceAccount) validateServiceAccountExists(client *kube.Client, result *verifier.Result) error {
	coreClient := client.KubeClient.CoreV1()
	saList, err := coreClient.ServiceAccounts(o.opts.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to retrieve list of service accounts from namespace %v: %v", o.opts.Namespace, err)
	}
	for _, sa := range saList.Items {
		if sa.Name == o.opts.ServiceAccount {
			clog.V(2).Printf("Service Account %s/%s exists", o.opts.Namespace, o.opts.ServiceAccount)
			return nil
		}
	}
	result.AddErrors(fmt.Sprintf("Service Account %s does not exists - KUDO expects the serviceAccount to be present in the namespace %s", o.opts.ServiceAccount, o.opts.Namespace))
	return nil
}

// Validate whether the serviceAccount has cluster-admin role
func (o KudoServiceAccount) validateClusterAdminRoleForSA(client *kube.Client, result *verifier.Result) error {
	// Check whether the serviceAccount has clusterrolebinding cluster-admin
	crbs, err := client.KubeClient.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to retrieve list of role bindings: %v", err)
	}

	for _, crb := range crbs.Items {
		for _, subject := range crb.Subjects {
			if subject.Name == o.opts.ServiceAccount && subject.Namespace == o.opts.Namespace && crb.RoleRef.Name == "cluster-admin" {
				clog.V(2).Printf("Service Account %s/%s has cluster-admin role", o.opts.Namespace, o.opts.ServiceAccount)
				return nil
			}
		}
	}

	result.AddErrors(fmt.Sprintf("Service Account %s does not have cluster-admin role - KUDO expects the serviceAccount passed to be in the namespace %s and to have cluster-admin role", o.opts.ServiceAccount, o.opts.Namespace))
	return nil
}

func (o KudoServiceAccount) installServiceAccount(client *kube.Client) error {
	coreClient := client.KubeClient.CoreV1()
	_, err := coreClient.ServiceAccounts(o.opts.Namespace).Create(context.TODO(), o.serviceAccount, metav1.CreateOptions{})
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("service account %v already exists", o.serviceAccount.Name)
		return nil
	}
	return err
}

func (o KudoServiceAccount) installRoleBinding(client *kube.Client) error {
	_, err := client.KubeClient.RbacV1().ClusterRoleBindings().Create(context.TODO(), o.roleBinding, metav1.CreateOptions{})
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("role binding %v already exists", o.roleBinding.Name)
		return nil
	}
	return err
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
