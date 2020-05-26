package prereq

import (
	"fmt"
	"strings"

	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientv1beta1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"

	"github.com/kudobuilder/kudo/pkg/engine/health"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
	"github.com/kudobuilder/kudo/pkg/util/convert"
)

// Ensure IF is implemented
var _ kudoinit.Step = &KudoWebHook{}

type KudoWebHook struct {
	opts        kudoinit.Options
	issuer      unstructured.Unstructured
	certificate unstructured.Unstructured
}

const (
	certManagerAPIVersion        = "v1alpha2"
	certManagerControllerVersion = "v0.12.0"
	instanceAdmissionWebHookName = "kudo-manager-instance-admission-webhook-config"
)

var (
	certManagerControllerImageSuffix = fmt.Sprintf("cert-manager-controller:%s", certManagerControllerVersion)
)

func NewWebHookInitializer(options kudoinit.Options) KudoWebHook {
	return KudoWebHook{
		opts:        options,
		certificate: certificate(options.Namespace),
		issuer:      issuer(options.Namespace),
	}
}

func (k KudoWebHook) String() string {
	return "webhook"
}

func (k KudoWebHook) PreInstallVerify(client *kube.Client, result *verifier.Result) error {
	// skip verification if webhooks are not used or self-signed CA is used
	if k.opts.SelfSignedWebhookCA {
		return nil
	}
	return validateCertManagerInstallation(client, result)
}

func (k KudoWebHook) PreUpgradeVerify(client *kube.Client, result *verifier.Result) error {
	// Nothing to verify here at the moment, needs to be extended when we have actual upgrades
	return nil
}

func (k KudoWebHook) VerifyInstallation(client *kube.Client, result *verifier.Result) error {
	if k.opts.SelfSignedWebhookCA {
		return nil
	}
	if err := validateCertManagerInstallation(client, result); err != nil {
		return err
	}
	if err := validateUnstructuredInstallation(client.DynamicClient, k.issuer, result); err != nil {
		return err
	}
	if err := validateUnstructuredInstallation(client.DynamicClient, k.certificate, result); err != nil {
		return err
	}
	if err := validateAdmissionWebhookInstallation(client.KubeClient.AdmissionregistrationV1beta1(), InstanceAdmissionWebhook(k.opts.Namespace), result); err != nil {
		return err
	}

	return nil
}

func (k KudoWebHook) installWithCertManager(client *kube.Client) error {
	if err := installUnstructured(client.DynamicClient, k.issuer); err != nil {
		return err
	}
	if err := installUnstructured(client.DynamicClient, k.certificate); err != nil {
		return err
	}
	if err := installAdmissionWebhook(client.KubeClient.AdmissionregistrationV1beta1(), InstanceAdmissionWebhook(k.opts.Namespace)); err != nil {
		return err
	}
	return nil
}

func (k KudoWebHook) installWithSelfSignedCA(client *kube.Client) error {
	iaw, s, err := k.resourcesWithSelfSignedCA()
	if err != nil {
		return nil
	}

	if err := installAdmissionWebhook(client.KubeClient.AdmissionregistrationV1beta1(), *iaw); err != nil {
		return err
	}

	if err := installWebhookSecret(client.KubeClient, *s); err != nil {
		return err
	}

	return nil
}

func (k KudoWebHook) Install(client *kube.Client) error {
	if k.opts.SelfSignedWebhookCA {
		return k.installWithSelfSignedCA(client)
	}
	return k.installWithCertManager(client)
}

func UninstallWebHook(client *kube.Client) error {
	err := client.KubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(instanceAdmissionWebHookName, &metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to uninstall WebHook: %v", err)
	}
	return nil
}

func (k KudoWebHook) Resources() []runtime.Object {
	if k.opts.SelfSignedWebhookCA {
		iaw, s, err := k.resourcesWithSelfSignedCA()
		if err != nil {
			panic(err)
		}
		return []runtime.Object{iaw, s}

	}

	return k.resourcesWithCertManager()
}

func (k KudoWebHook) resourcesWithCertManager() []runtime.Object {
	av := InstanceAdmissionWebhook(k.opts.Namespace)
	objs := []runtime.Object{&av}
	objs = append(objs, &k.issuer)
	objs = append(objs, &k.certificate)
	return objs
}

func (k KudoWebHook) resourcesWithSelfSignedCA() (*admissionv1beta1.MutatingWebhookConfiguration, *corev1.Secret, error) {
	tinyCA, err := NewTinyCA(kudoinit.DefaultServiceName, k.opts.Namespace)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to set up webhook CA: %v", err)
	}

	srvCertPair, err := tinyCA.NewServingCert()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to set up webhook serving certs: %v", err)
	}

	srvCert, srvKey, err := srvCertPair.AsBytes()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to marshal webhook serving certs: %v", err)
	}

	iaw := instanceAdmissionWebhookWithCABundle(k.opts.Namespace, tinyCA.CA.CertBytes())
	ws := webhookSecret(k.opts.Namespace, srvCert, srvKey)

	return &iaw, &ws, nil
}

func validateCertManagerInstallation(client *kube.Client, result *verifier.Result) error {
	if err := validateCrdVersion(client.ExtClient, "certificates.cert-manager.io", certManagerAPIVersion, result); err != nil {
		return err
	}
	if err := validateCrdVersion(client.ExtClient, "issuers.cert-manager.io", certManagerAPIVersion, result); err != nil {
		return err
	}

	if !result.IsEmpty() {
		// Abort verify here, if we don't have CRDs the remaining checks don't make much sense
		return nil
	}

	deployment, err := client.KubeClient.AppsV1().Deployments("cert-manager").Get("cert-manager", metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			result.AddWarnings(fmt.Sprintf("failed to get cert-manager deployment in namespace cert-manager. Make sure cert-manager is running."))
			return nil
		}
		return err
	}
	if len(deployment.Spec.Template.Spec.Containers) < 1 {
		result.AddWarnings("failed to validate cert-manager controller deployment. Spec had no containers")
		return nil
	}
	if !strings.HasSuffix(deployment.Spec.Template.Spec.Containers[0].Image, certManagerControllerImageSuffix) {
		result.AddWarnings(fmt.Sprintf("cert-manager deployment had unexpected version. expected %s in controller image name but found %s", certManagerControllerVersion, deployment.Spec.Template.Spec.Containers[0].Image))
		return nil
	}
	if err := health.IsHealthy(deployment); err != nil {
		result.AddWarnings("cert-manager seems not to be running correctly. Make sure cert-manager is working")
		return nil
	}
	return nil
}

func validateCrdVersion(extClient clientset.Interface, crdName string, expectedVersion string, result *verifier.Result) error {
	certCRD, err := extClient.ApiextensionsV1().CustomResourceDefinitions().Get(crdName, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			result.AddErrors(fmt.Sprintf("failed to find CRD '%s': %s", crdName, err))
			return nil
		}
		return err
	}
	crdVersion := certCRD.Spec.Versions[0].Name

	if crdVersion != expectedVersion {
		result.AddErrors(fmt.Sprintf("invalid CRD version found for '%s': %s instead of %s", crdName, crdVersion, expectedVersion))
	}
	return nil
}

// installUnstructured accepts kubernetes resource as unstructured.Unstructured and installs it into cluster
func installUnstructured(dynamicClient dynamic.Interface, item unstructured.Unstructured) error {
	gvk := item.GroupVersionKind()
	_, err := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: fmt.Sprintf("%ss", strings.ToLower(gvk.Kind)), // since we know what kinds are we dealing with here, this is OK
	}).Namespace(item.GetNamespace()).Create(&item, metav1.CreateOptions{})
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("resource %s already registered", item.GetName())
	} else if err != nil {
		return fmt.Errorf("error when creating resource %s/%s. %v", item.GetName(), item.GetNamespace(), err)
	}
	return nil
}

func validateUnstructuredInstallation(dynamicClient dynamic.Interface, item unstructured.Unstructured, result *verifier.Result) error {
	gvk := item.GroupVersionKind()
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: fmt.Sprintf("%ss", strings.ToLower(gvk.Kind)), // since we know what kinds are we dealing with here, this is OK
	}

	_, err := dynamicClient.Resource(gvr).Namespace(item.GetNamespace()).Get(item.GetName(), metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			result.AddErrors(fmt.Sprintf("%s is not installed in namespace %s", item.GetName(), item.GetNamespace()))
			return nil
		}
		return err
	}

	// We could add more detailed validation here, but DeepEquals doesn't work because of added fields from k8s

	return nil
}

func installAdmissionWebhook(client clientv1beta1.MutatingWebhookConfigurationsGetter, webhook admissionv1beta1.MutatingWebhookConfiguration) error {
	_, err := client.MutatingWebhookConfigurations().Create(&webhook)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("admission webhook %v already registered", webhook.Name)
		return nil
	}
	return err
}

func installWebhookSecret(client kubernetes.Interface, secret corev1.Secret) error {
	_, err := client.CoreV1().Secrets(secret.Namespace).Create(&secret)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("webhook secret %v already exists", secret.Name)
		return nil
	}
	return err
}

func instanceAdmissionWebhookWithCABundle(ns string, caData []byte) admissionv1beta1.MutatingWebhookConfiguration {
	iaw := InstanceAdmissionWebhook(ns)
	iaw.Webhooks[0].ClientConfig.CABundle = caData
	return iaw
}

func validateAdmissionWebhookInstallation(client clientv1beta1.MutatingWebhookConfigurationsGetter, webhook admissionv1beta1.MutatingWebhookConfiguration, result *verifier.Result) error {
	_, err := client.MutatingWebhookConfigurations().Get(webhook.Name, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			result.AddErrors(fmt.Sprintf("admission webhook %s is not installed", webhook.Name))
			return nil
		}
		return err
	}

	// We could add more detailed validation here, regarding the details of the webhook configuration

	return nil
}

// InstanceAdmissionWebhook returns a MutatingWebhookConfiguration for the instance admission controller.
func InstanceAdmissionWebhook(ns string) admissionv1beta1.MutatingWebhookConfiguration {
	namespacedScope := admissionv1beta1.NamespacedScope
	failedType := admissionv1beta1.Fail
	equivalentType := admissionv1beta1.Equivalent
	noSideEffects := admissionv1beta1.SideEffectClassNone
	return admissionv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: instanceAdmissionWebHookName,
			Annotations: map[string]string{
				"cert-manager.io/inject-ca-from": fmt.Sprintf("%s/kudo-webhook-server-certificate", ns),
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "MutatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1beta1",
		},
		Webhooks: []admissionv1beta1.MutatingWebhook{
			{
				Name: "instance-admission.kudo.dev",
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
						Name:      kudoinit.DefaultServiceName,
						Namespace: ns,
						Path:      convert.StringPtr("/admit-kudo-dev-v1beta1-instance"),
					},
				},
			},
		},
	}
}

func issuer(ns string) unstructured.Unstructured {
	return unstructured.Unstructured{
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
	}
}

func certificate(ns string) unstructured.Unstructured {
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cert-manager.io/v1alpha2",
			"kind":       "Certificate",
			"metadata": map[string]interface{}{
				"name":      "kudo-webhook-server-certificate",
				"namespace": ns,
			},
			"spec": map[string]interface{}{
				"commonName": fmt.Sprintf("kudo-controller-manager-service.%s.svc", ns),
				"dnsNames":   []string{fmt.Sprintf("kudo-controller-manager-service.%s.svc", ns)},
				"issuerRef": map[string]interface{}{
					"kind": "Issuer",
					"name": "selfsigned-issuer",
				},
				"secretName": kudoinit.DefaultSecretName,
			},
		},
	}
}

func webhookSecret(ns string, cert, key []byte) corev1.Secret {
	return corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kudoinit.DefaultSecretName,
			Namespace: ns,
		},
		Type: "kubernetes.io/tls",
		Data: map[string][]byte{
			"tls.crt": cert,
			"tls.key": key,
		},
	}
}
