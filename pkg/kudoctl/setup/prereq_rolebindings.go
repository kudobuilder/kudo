package setup

import (
	"fmt"
	"reflect"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"

	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type KudoRoleBinding struct {
	opts Options
	crb  *rbacv1.ClusterRoleBinding
}

func newRoleBindingSetup(options Options) KudoRoleBinding {
	return KudoRoleBinding{
		opts: options,
		crb:  generateRoleBinding(options),
	}
}

func (o KudoRoleBinding) Install(client *kube.Client) error {
	_, err := client.KubeClient.RbacV1().ClusterRoleBindings().Create(o.crb)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("role binding %v already exists", o.crb.Name)
		return nil
	}
	return err
}

func (o KudoRoleBinding) Validate(client *kube.Client) error {
	existing, err := client.KubeClient.RbacV1().RoleBindings(o.opts.Namespace).Get(o.crb.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to retrieve role binding %v", err)
	}

	if !reflect.DeepEqual(existing, o.crb) {
		return fmt.Errorf("installed ClusterRoleBinding does not equal expected")
	}
	return nil
}

func (o KudoRoleBinding) AsRuntimeObj() []runtime.Object {
	return []runtime.Object{o.crb}
}

func generateRoleBinding(opts Options) *rbacv1.ClusterRoleBinding {
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
