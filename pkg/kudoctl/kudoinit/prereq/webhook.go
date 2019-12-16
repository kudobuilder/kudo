package prereq

import (
	"fmt"
	"strings"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/util/kudo"

	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	clientv1beta1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"
)

// Ensure IF is implemented
var _ k8sResource = &kudoWebHook{}

type kudoWebHook struct {
	opts kudoinit.Options
}

const (
	certManagerAPIVersion = "cert-manager.io/v1alpha2"
)

func newWebHook(options kudoinit.Options) kudoWebHook {
	return kudoWebHook{
		opts: options,
	}
}

func (k kudoWebHook) PreInstallCheck(client *kube.Client) error {
	if err := validateCertManagerInstallation(client.ExtClient); err != nil {
		return err
	}
	return nil
}

func (k kudoWebHook) Install(client *kube.Client) error {
	if !k.opts.HasWebhooksEnabled() {
		return nil
	}

	if err := installUnstructured(client.DynamicClient, certificate(k.opts.Namespace)); err != nil {
		return err
	}
	if err := installAdmissionWebhook(client.KubeClient.AdmissionregistrationV1beta1(), instanceUpdateValidatingWebhook(k.opts.Namespace)); err != nil {
		return err
	}
	return nil
}

func (k kudoWebHook) ValidateInstallation(client *kube.Client) error {
	if !k.opts.HasWebhooksEnabled() {
		return nil
	}

	// TODO: Check installed webhooks, check if cert-manager is installed, etc
	panic("implement me")
}

func (k kudoWebHook) AsRuntimeObjs() []runtime.Object {
	if !k.opts.HasWebhooksEnabled() {
		return make([]runtime.Object, 0)
	}

	av := instanceUpdateValidatingWebhook(k.opts.Namespace)
	cert := certificate(k.opts.Namespace)
	objs := []runtime.Object{&av}
	for _, c := range cert {
		c := c
		objs = append(objs, &c)
	}
	return objs
}

func validateCertManagerInstallation(extClient apiextensionsclient.Interface) error {
	certCRD, err := extClient.ApiextensionsV1().CustomResourceDefinitions().Get("Certificate", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("cert-manager is not installed: %s", err)
	}
	if certCRD.APIVersion != certManagerAPIVersion {
		return fmt.Errorf("invalid cert-manager API version found for Certificate: %s instead of %s", certCRD.APIVersion, certManagerAPIVersion)
	}

	issuerCRD, err := extClient.ApiextensionsV1().CustomResourceDefinitions().Get("Issuer", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("cert-manager is not installed: %s", err)
	}
	if issuerCRD.APIVersion != certManagerAPIVersion {
		return fmt.Errorf("invalid cert-manager API version found for Issuer: %s instead of %s", issuerCRD, certManagerAPIVersion)
	}

	return nil
}

// installUnstructured accepts kubernetes resource as unstructured.Unstructured and installs it into cluster
func installUnstructured(dynamicClient dynamic.Interface, items []unstructured.Unstructured) error {
	for _, item := range items {
		obj := item
		gvk := item.GroupVersionKind()
		_, err := dynamicClient.Resource(schema.GroupVersionResource{
			Group:    gvk.Group,
			Version:  gvk.Version,
			Resource: fmt.Sprintf("%ss", strings.ToLower(gvk.Kind)), // since we know what kinds are we dealing with here, this is OK
		}).Namespace(obj.GetNamespace()).Create(&obj, metav1.CreateOptions{})
		if kerrors.IsAlreadyExists(err) {
			clog.V(4).Printf("resource %s already registered", obj.GetName())
		} else if err != nil {
			return fmt.Errorf("Error when creating resource %s/%s. %v", obj.GetName(), obj.GetNamespace(), err)
		}
	}
	return nil
}

func installAdmissionWebhook(client clientv1beta1.ValidatingWebhookConfigurationsGetter, webhook admissionv1beta1.ValidatingWebhookConfiguration) error {
	_, err := client.ValidatingWebhookConfigurations().Create(&webhook)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("admission webhook %v already registered", webhook.Name)
		return nil
	}
	return err
}

func instanceUpdateValidatingWebhook(ns string) admissionv1beta1.ValidatingWebhookConfiguration {
	namespacedScope := admissionv1beta1.NamespacedScope
	failedType := admissionv1beta1.Fail
	equivalentType := admissionv1beta1.Equivalent
	noSideEffects := admissionv1beta1.SideEffectClassNone
	return admissionv1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kudo-manager-instance-validation-webhook-config",
			Annotations: map[string]string{
				"cert-manager.io/inject-ca-from": fmt.Sprintf("%s/kudo-webhook-server-certificate", ns),
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ValidatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1beta1",
		},
		Webhooks: []admissionv1beta1.ValidatingWebhook{
			{
				Name: "instance-validation.kudo.dev",
				Rules: []admissionv1beta1.RuleWithOperations{
					{
						Operations: []admissionv1beta1.OperationType{"CREATE", "UPDATE"},
						Rule: admissionv1beta1.Rule{
							APIGroups:   []string{"kudo.dev"},
							APIVersions: []string{"v1beta1"},
							Resources:   []string{"instances"},
							Scope:       &namespacedScope,
						},
					},
				},
				FailurePolicy: &failedType, // this means that the request to update instance would fail, if webhook is not up
				MatchPolicy:   &equivalentType,
				SideEffects:   &noSideEffects,
				ClientConfig: admissionv1beta1.WebhookClientConfig{
					Service: &admissionv1beta1.ServiceReference{
						Name:      "kudo-controller-manager-service",
						Namespace: ns,
						Path:      kudo.String("/validate-kudo-dev-v1beta1-instance"),
					},
				},
			},
		},
	}
}

func certificate(ns string) []unstructured.Unstructured {
	return []unstructured.Unstructured{
		{
			Object: map[string]interface{}{
				"apiVersion": "cert-manager.io/v1alpha2",
				"kind":       "Issuer",
				"metadata": map[string]interface{}{
					"name":      "selfsigned-issuer",
					"namespace": ns,
				},
				"spec": map[string]interface{}{
					"selfSigned": map[string]interface{}{},
				},
			},
		},
		{
			Object: map[string]interface{}{
				"apiVersion": "cert-manager.io/v1alpha2",
				"kind":       "Certificate",
				"metadata": map[string]interface{}{
					"name":      "kudo-webhook-server-certificate",
					"namespace": ns,
				},
				"spec": map[string]interface{}{
					"commonName": "kudo-controller-manager-service.kudo-system.svc",
					"dnsNames":   []string{"kudo-controller-manager-service.kudo-system.svc"},
					"issuerRef": map[string]interface{}{
						"kind": "Issuer",
						"name": "selfsigned-issuer",
					},
					"secretName": "kudo-webhook-server-secret",
				},
			},
		},
	}
}
