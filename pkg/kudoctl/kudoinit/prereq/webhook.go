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

type certManagerVersion struct {
	group    string
	versions []string
}

var (
	// Cert-Manager APIs that we can detect
	certManagerAPIs = []certManagerVersion{
		{group: "cert-manager.io", versions: []string{"v1alpha2", "v1alpha3"}}, // 0.11.0+
		{group: "certmanager.k8s.io", versions: []string{"v1alpha1"}},          // 0.10.1
	}
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
	if err := installAdmissionWebhook(client.KubeClient.AdmissionregistrationV1beta1(), instanceAdmissionWebhookCertManager(k.opts.Namespace, k.certManagerGroup)); err != nil {
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
	av := instanceAdmissionWebhookCertManager(k.opts.Namespace, k.certManagerGroup)
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

	for _, api := range certManagerAPIs {
		group, version, err := detectCertManagerCRD(extClient, api)
		if err != nil {
			return err
		}
		if group != "" && version != "" {
			clog.V(4).Printf("Detected cert-manager CRDs %s/%s", k.certManagerGroup, k.certManagerAPIVersion)

			if !funk.Contains(api.versions, version) {
				result.AddWarnings(fmt.Sprintf("Detected cert-manager CRDs with version %s, only versions %v are fully supported. Certificates for webhooks may not work.", version, api.versions))
			}

			k.certManagerGroup = group
			k.certManagerAPIVersion = version

			return nil
		}
	}

	result.AddErrors(fmt.Sprintf("failed to detect any valid cert-manager CRDs. Make sure cert-manager is installed."))
	return nil
}

func detectCertManagerCRD(extClient clientset.Interface, api certManagerVersion) (string, string, error) {
	testCRD := fmt.Sprintf("certificates.%s", api.group)
	clog.V(4).Printf("Try to retrieve cert-manager CRD %s", testCRD)
	crd, err := extClient.ApiextensionsV1().CustomResourceDefinitions().Get(testCRD, metav1.GetOptions{})
	if err == nil {
		// crd.Spec.Versions[0] must be the one that is stored and served, we should use that one
		clog.V(4).Printf("Got CRD. Group: %s, Version: %s", api.group, crd.Spec.Versions[0].Name)
		return api.group, crd.Spec.Versions[0].Name, nil
	}
	if !kerrors.IsNotFound(err) {
		return "", "", fmt.Errorf("failed to detect cert manager CRD %s: %v", testCRD, err)
	}
	return "", "", nil
}

func (k *KudoWebHook) validateCertManagerInstallation(client *kube.Client, result *verifier.Result) error {
	if err := k.detectCertManagerVersion(client, result); err != nil {
		return err
	}
	if !result.IsValid() {
		return nil
	}

	certificateCRD := fmt.Sprintf("certificates.%s", k.certManagerGroup)
	if err := validateCrdVersion(client.ExtClient, certificateCRD, k.certManagerAPIVersion, result); err != nil {
		return err
	}
	issuerCRD := fmt.Sprintf("issuers.%s", k.certManagerGroup)
	if err := validateCrdVersion(client.ExtClient, issuerCRD, k.certManagerAPIVersion, result); err != nil {
		return err
	}

	if !result.IsEmpty() {
		// Abort verify here, if we don't have CRDs the remaining checks don't make much sense
		return nil
	}

	// Initialize the custom resources that we're going to install
	k.certificate = certificate(k.opts.Namespace, k.certManagerGroup, k.certManagerAPIVersion)
	k.issuer = issuer(k.opts.Namespace, k.certManagerGroup, k.certManagerAPIVersion)

	// A couple extra checks, checking for cert manager, detection requires the label app=cert-manager which is the
	// default according to k8s.io docs.
	deployments, err := client.KubeClient.AppsV1().Deployments("").List(metav1.ListOptions{
		LabelSelector: "app=cert-manager",
	})
	if err != nil {
		// err is an infra error, 0 deploys is not an error
		return err
	}
	switch cnt := len(deployments.Items); {
	case cnt == 0:
		result.AddWarnings(fmt.Sprintf("unable to find cert-manager deployment. Make sure cert-manager is running."))
		return nil
	case cnt > 1:
		result.AddWarnings(fmt.Sprintf("more than 1 cert-manager deployment found."))
	}

	// for some reason the list of objects (which are []Deployment) are stripped of their kind and apiversions (causing issues with unstructuring in the isHealth func)
	// there should only be 1, regardless we check the first (the warning for more than 1 found is already provided above)
	deployment := deployments.Items[0]
	deployment.Kind = "Deployment"
	deployment.APIVersion = "apps/v1"

	if len(deployment.Spec.Template.Spec.Containers) < 1 {
		result.AddWarnings("unable to validate cert-manager controller deployment. Spec had no containers")
		return nil
	}
	if err := health.IsHealthy(&deployment); err != nil {
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

func instanceAdmissionWebhookCertManager(ns string, certManagerGroup string) admissionv1beta1.MutatingWebhookConfiguration {
	iaw := InstanceAdmissionWebhook(ns)
	injectCaAnnotationName := fmt.Sprintf("%s/inject-ca-from", certManagerGroup)
	iaw.Annotations[injectCaAnnotationName] = fmt.Sprintf("%s/kudo-webhook-server-certificate", ns)
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
			Name:        "kudo-manager-instance-admission-webhook-config",
			Annotations: map[string]string{},
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
