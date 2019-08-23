package init

import (
	"github.com/ghodss/yaml"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	extensionsclient "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
)

const namespace = "kudo-system"

// Install uses Kubernetes client to install KUDO manager.
func Install(client kubernetes.Interface) error {
	if err := createDeployment(client.ExtensionsV1beta1()); err != nil {
		return err
	}

	if err := createService(client.CoreV1(), ""); err != nil {
		return err
	}
	if err := createSecret(client.CoreV1()); err != nil {
		return err
	}
	return nil
}

func createService(client corev1.ServicesGetter, namespace string) error {
	obj := generateManagerService(namespace)
	_, err := client.Services(obj.Namespace).Create(obj)
	return err

}

// createDeployment creates the KUDO manager Deployment resource.
func createDeployment(client extensionsclient.DeploymentsGetter) error {
	obj, err := generateManagerDeployment()
	if err != nil {
		return err
	}
	_, err = client.Deployments(obj.Namespace).Create(obj)
	return err
}

// createSecret creates the KUDO secret resource.
func createSecret(client corev1.SecretsGetter) error {
	o := generateWebHookSecret()
	_, err := client.Secrets(o.Namespace).Create(o)
	return err
}

func generateManagerService(namespace string) *v1.Service {
	return nil
}

func generateManagerDeployment() (*v1beta1.Deployment, error) {
	return nil, nil
}

// generateSysNamespace builds the system namespace
func generateSysNamespace() *v1.Namespace {
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
func generateServiceAccount() *v1.ServiceAccount {
	labels := generateLabels(map[string]string{})
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      "kudo-manager",
			Namespace: namespace,
		},
	}

	return sa
}

// generateRoleBinding builds the cluster role binding
func generateRoleBinding() *rbacv1.ClusterRoleBinding {
	sa := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kudo-manager-rolebinding",
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{rbacv1.Subject{
			Kind:      "ServiceAccount",
			Name:      "kudo-manager",
			Namespace: namespace,
		}},
	}
	return sa
}

// generateWebHookSecret builds the secret object used for webhooks
func generateWebHookSecret() *v1.Secret {
	secret := &v1.Secret{
		Data: make(map[string][]byte),
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kudo-webhook-server-secret",
			Namespace: namespace,
		},
	}

	return secret
}

func generateLabels(labels map[string]string) map[string]string {
	labels["app"] = "kudo-manager"
	return labels
}

// PrereqManifests provides a slice of strings for each pre requisite manifest
func PrereqManifests() ([]string, error) {
	ns := Namespace()
	svc := ServiceAccount()
	rbac := RoleBinding()
	secret := WebhookSecret()

	objs := []runtime.Object{ns, svc, rbac, secret}

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

// RoleBinding provides the RoleBinding rbac manifest for printing
func RoleBinding() *rbacv1.ClusterRoleBinding {
	rbac := generateRoleBinding()
	rbac.TypeMeta = metav1.TypeMeta{
		Kind:       "ClusterRoleBinding",
		APIVersion: "rbac.authorization.k8s.io/v1",
	}
	return rbac
}

// WebhookSecret provides the webhook secret manifest for printing
func WebhookSecret() *v1.Secret {
	secret := generateWebHookSecret()
	secret.TypeMeta = metav1.TypeMeta{
		Kind:       "Secret",
		APIVersion: "v1",
	}
	return secret
}

// ServiceAccount provides the service account manifest for printing
func ServiceAccount() *v1.ServiceAccount {
	sa := generateServiceAccount()
	sa.TypeMeta = metav1.TypeMeta{
		Kind:       "ServiceAccount",
		APIVersion: "v1",
	}
	return sa
}

// Namespace provides the namespace manifest for printing
func Namespace() *v1.Namespace {
	ns := generateSysNamespace()
	ns.TypeMeta = metav1.TypeMeta{
		Kind:       "Namespace",
		APIVersion: "v1",
	}
	return ns
}
