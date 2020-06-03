package prereq

import (
	"fmt"
	"strings"

	"github.com/thoas/go-funk"
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
	opts kudoinit.Options

	certManagerGroup      string
	certManagerAPIVersion string

	issuer      *unstructured.Unstructured
	certificate *unstructured.Unstructured
}

const (
	certManagerOldGroup = "certmanager.k8s.io"
	certManagerNewGroup = "cert-manager.io"
)

var (
	certManagerAPIVersions = []string{"v1alpha2", "v1alpha1"}
)

func NewWebHookInitializer(options kudoinit.Options) *KudoWebHook {
	return &KudoWebHook{
		opts: options,
	}
}

func (k *KudoWebHook) String() string {
	return "webhook"
}

func (k *KudoWebHook) PreInstallVerify(client *kube.Client, result *verifier.Result) error {
	// skip verification if webhooks are not used or self-signed CA is used
	if k.opts.SelfSignedWebhookCA {
		return nil
	}
	return k.validateCertManagerInstallation(client, result)
}

func (k *KudoWebHook) installWithCertManager(client *kube.Client) error {
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

func (k *KudoWebHook) installWithSelfSignedCA(client *kube.Client) error {
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

func (k *KudoWebHook) Install(client *kube.Client) error {
	if k.opts.SelfSignedWebhookCA {
		return k.installWithSelfSignedCA(client)
	}
	return k.installWithCertManager(client)
}

func (k *KudoWebHook) Resources() []runtime.Object {
	if k.opts.SelfSignedWebhookCA {
		iaw, s, err := k.resourcesWithSelfSignedCA()
		if err != nil {
			panic(err)
		}
		return []runtime.Object{iaw, s}

	}

	return k.resourcesWithCertManager()
}

func (k *KudoWebHook) resourcesWithCertManager() []runtime.Object {
	// We have to fall back to a default here as for a dry-run we can't detect the actual version of a cluster
	k.issuer = issuer(k.opts.Namespace, certManagerNewGroup, certManagerAPIVersions[0])
	k.certificate = certificate(k.opts.Namespace, certManagerNewGroup, certManagerAPIVersions[0])

	av := InstanceAdmissionWebhook(k.opts.Namespace)
	objs := []runtime.Object{&av}
	objs = append(objs, k.issuer)
	objs = append(objs, k.certificate)
	return objs
}

func (k *KudoWebHook) resourcesWithSelfSignedCA() (*admissionv1beta1.MutatingWebhookConfiguration, *corev1.Secret, error) {
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

func (k *KudoWebHook) detectCertManagerVersion(client *kube.Client, result *verifier.Result) error {
	extClient := client.ExtClient

	// Cert-Manager 0.11.0+
	if err := k.detectCertManagerCRD(extClient, certManagerNewGroup); err != nil {
		return err
	}
	if k.certManagerGroup != "" {
		return nil
	}

	// Cert-Manager 0.10.1
	if err := k.detectCertManagerCRD(extClient, certManagerOldGroup); err != nil {
		return err
	}
	if k.certManagerGroup != "" {
		return nil
	}
	result.AddErrors(fmt.Sprintf("failed to detect any valid cert-manager CRDs. Make sure cert-manager is installed."))
	return nil
}

func (k *KudoWebHook) detectCertManagerCRD(extClient clientset.Interface, group string) error {
	testCRD := fmt.Sprintf("certificates.%s", group)
	clog.V(4).Printf("Try to retrieve cert-manager CRD %s", testCRD)
	crd, err := extClient.ApiextensionsV1().CustomResourceDefinitions().Get(testCRD, metav1.GetOptions{})
	if err == nil {
		clog.V(4).Printf("Got CRD. Group: %s, Version: %s", group, crd.Spec.Versions[0].Name)
		k.certManagerGroup = group
		k.certManagerAPIVersion = crd.Spec.Versions[0].Name
		return nil
	}
	if !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to detect cert manager CRD %s: %v", testCRD, err)
	}
	return nil
}

func (k *KudoWebHook) validateCertManagerInstallation(client *kube.Client, result *verifier.Result) error {
	if err := k.detectCertManagerVersion(client, result); err != nil {
		return err
	}
	if !result.IsValid() {
		return nil
	}
	clog.V(4).Printf("Detected cert-manager CRDs %s/%s", k.certManagerGroup, k.certManagerAPIVersion)

	if !funk.Contains(certManagerAPIVersions, k.certManagerAPIVersion) {
		result.AddWarnings(fmt.Sprintf("Detected cert-manager CRDs with version %s, only versions %v are fully supported. Certificates for webhooks may not work.", k.certManagerAPIVersion, certManagerAPIVersions))
	}

	k.certificate = certificate(k.opts.Namespace, k.certManagerGroup, k.certManagerAPIVersion)
	k.issuer = issuer(k.opts.Namespace, k.certManagerGroup, k.certManagerAPIVersion)

	certificateCRD := fmt.Sprintf("certificates.%s", k.certManagerGroup)
	if err := validateCrdVersion(client.ExtClient, certificateCRD, k.certificate.GroupVersionKind().Version, result); err != nil {
		return err
	}
	issuerCRD := fmt.Sprintf("issuers.%s", k.certManagerGroup)
	if err := validateCrdVersion(client.ExtClient, issuerCRD, k.issuer.GroupVersionKind().Version, result); err != nil {
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
func installUnstructured(dynamicClient dynamic.Interface, item *unstructured.Unstructured) error {
	gvk := item.GroupVersionKind()
	_, err := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: fmt.Sprintf("%ss", strings.ToLower(gvk.Kind)), // since we know what kinds are we dealing with here, this is OK
	}).Namespace(item.GetNamespace()).Create(item, metav1.CreateOptions{})
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("resource %s already registered", item.GetName())
	} else if err != nil {
		return fmt.Errorf("error when creating resource %s/%s. %v", item.GetName(), item.GetNamespace(), err)
	}
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

// InstanceAdmissionWebhook returns a MutatingWebhookConfiguration for the instance admission controller.
func InstanceAdmissionWebhook(ns string) admissionv1beta1.MutatingWebhookConfiguration {
	namespacedScope := admissionv1beta1.NamespacedScope
	failedType := admissionv1beta1.Fail
	equivalentType := admissionv1beta1.Equivalent
	noSideEffects := admissionv1beta1.SideEffectClassNone
	return admissionv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kudo-manager-instance-admission-webhook-config",
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

func issuer(ns string, group string, apiVersion string) *unstructured.Unstructured {
	apiString := fmt.Sprintf("%s/%s", group, apiVersion)
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiString,
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

func certificate(ns string, group string, apiVersion string) *unstructured.Unstructured {
	apiString := fmt.Sprintf("%s/%s", group, apiVersion)
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiString,
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
