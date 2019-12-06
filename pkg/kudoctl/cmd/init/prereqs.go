package init

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
)

//Defines the Prerequisites that need to be in place to run the KUDO manager.  This includes setting up the kudo-system namespace and service account

// Install uses Kubernetes client to install KUDO manager prereqs.
func installPrereqs(client kubernetes.Interface, opts Options) error {
	if err := installNamespace(client.CoreV1(), opts); err != nil {
		return err
	}

	if err := installServiceAccount(client.CoreV1(), opts); err != nil {
		return err
	}
	if err := installRoleBindings(client, opts); err != nil {
		return err
	}
	if err := installSecret(client.CoreV1(), opts); err != nil {
		return err
	}
	return nil
}

func installSecret(client corev1.SecretsGetter, opts Options) error {
	secret := generateWebHookSecret(opts)
	_, err := client.Secrets(opts.Namespace).Create(secret)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("secret %v already exists", secret.Name)
		return nil
	}
	return err
}

func installRoleBindings(client kubernetes.Interface, opts Options) error {
	rbac := generateRoleBinding(opts)
	_, err := client.RbacV1().ClusterRoleBindings().Create(rbac)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("role binding %v already exists", rbac.Name)
		return nil
	}
	return err
}

func installNamespace(client corev1.NamespacesGetter, opts Options) error {
	// We only manage kudo-system namespace. For others we expect they exist.
	if opts.Namespace != defaultns {
		_, err := client.Namespaces().Get(opts.Namespace, metav1.GetOptions{})
		if err == nil {
			return nil
		}
		if kerrors.IsNotFound(err) {
			return fmt.Errorf("namespace %s does not exist - KUDO expects that any namespace except the default %s is created beforehand", opts.Namespace, defaultns)
		}
		return err
	}

	ns := generateSysNamespace(opts.Namespace)
	_, err := client.Namespaces().Create(ns)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("namespace %v already exists", ns.Name)
		return nil
	}
	return err
}

func installServiceAccount(client corev1.ServiceAccountsGetter, opts Options) error {
	sa := generateServiceAccount(opts)
	_, err := client.ServiceAccounts(opts.Namespace).Create(sa)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("service account %v already exists", sa.Name)
		return nil
	}
	return err
}

// generateSysNamespace builds the system namespace
func generateSysNamespace(namespace string) *v1.Namespace {
	labels := generateLabels(map[string]string{"controller-tools.k8s.io": "1.0"})
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
			Name:   namespace,
		},
	}

	return ns
}

// generateServiceAccount builds the system account
func generateServiceAccount(opts Options) *v1.ServiceAccount {
	labels := generateLabels(map[string]string{})
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      "kudo-manager",
			Namespace: opts.Namespace,
		},
	}

	return sa
}

// generateRoleBinding builds the cluster role binding
func generateRoleBinding(opts Options) *rbacv1.ClusterRoleBinding {
	sa := &rbacv1.ClusterRoleBinding{
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
			Name:      "kudo-manager",
			Namespace: opts.Namespace,
		}},
	}
	return sa
}

// generateWebHookSecret builds the secret object used for webhooks
func generateWebHookSecret(opts Options) *v1.Secret {
	secret := &v1.Secret{
		Data: make(map[string][]byte),
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kudo-webhook-server-secret",
			Namespace: opts.Namespace,
		},
	}

	return secret
}

func generateLabels(labels map[string]string) map[string]string {
	labels["app"] = "kudo-manager"
	return labels
}

// PrereqManifests provides a slice of strings for each pre requisite manifest
func PrereqManifests(opts Options) ([]string, error) {
	objs := Prereq(opts)
	manifests := make([]string, len(objs))
	for i, obj := range objs {
		o, err := yaml.Marshal(obj)
		if err != nil {
			return []string{}, err
		}
		manifests[i] = string(o)
	}

	return manifests, nil
}

// Prereq returns the slice of prerequisite objects for KUDO
func Prereq(opts Options) []runtime.Object {
	var prereqs []runtime.Object

	// We only manage kudo-system namespace. For others we expect they exist.
	if opts.Namespace == defaultns {
		prereqs = append(prereqs, namespace(opts.Namespace))
	}

	return append(
		prereqs,
		serviceAccount(opts),
		roleBinding(opts),
		webhookSecret(opts),
	)
}

// roleBinding provides the roleBinding rbac manifest for printing
func roleBinding(opts Options) *rbacv1.ClusterRoleBinding {
	rbac := generateRoleBinding(opts)
	rbac.TypeMeta = metav1.TypeMeta{
		Kind:       "ClusterRoleBinding",
		APIVersion: "rbac.authorization.k8s.io/v1",
	}
	return rbac
}

// webhookSecret provides the webhook secret manifest for printing
func webhookSecret(opts Options) *v1.Secret {
	secret := generateWebHookSecret(opts)
	secret.TypeMeta = metav1.TypeMeta{
		Kind:       "Secret",
		APIVersion: "v1",
	}
	return secret
}

// serviceAccount provides the service account manifest for printing
func serviceAccount(opts Options) *v1.ServiceAccount {
	sa := generateServiceAccount(opts)
	sa.TypeMeta = metav1.TypeMeta{
		Kind:       "ServiceAccount",
		APIVersion: "v1",
	}
	return sa
}

// namespace provides the namespace manifest for printing
func namespace(namespace string) *v1.Namespace {
	ns := generateSysNamespace(namespace)
	ns.TypeMeta = metav1.TypeMeta{
		Kind:       "Namespace",
		APIVersion: "v1",
	}
	return ns
}
