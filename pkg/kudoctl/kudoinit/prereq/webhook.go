package prereq

import (
	"fmt"
	"strings"

	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
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
	certManagerAPIVersion         = "v1alpha2"
	certManagerControllerVersion  = "v0.12.0"
	instanceValidationWebHookName = "kudo-manager-instance-validation-webhook-config"
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
	if !k.opts.HasWebhooksEnabled() {
		return nil
	}
	return validateCertManagerInstallation(client, result)
}

func (k KudoWebHook) PreUpgradeVerify(client *kube.Client, result *verifier.Result) error {
	return nil
}

func (k KudoWebHook) VerifyInstallation(client *kube.Client, result *verifier.Result) error {
	if !k.opts.HasWebhooksEnabled() {
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

	// TODO: Verify that validating webhook is installed

	return nil
}

func (k KudoWebHook) Install(client *kube.Client) error {
	if !k.opts.HasWebhooksEnabled() {
		return nil
	}

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

func UninstallWebHook(client *kube.Client, options kudoinit.Options) error {
	err := client.KubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(instanceValidationWebHookName, &metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to uninstall WebHook: %v", err)
	}
	return nil
}

func (k KudoWebHook) Resources() []runtime.Object {
	if !k.opts.HasWebhooksEnabled() {
		return make([]runtime.Object, 0)
	}

	av := InstanceAdmissionWebhook(k.opts.Namespace)
	objs := []runtime.Object{&av}
	objs = append(objs, &k.issuer)
	objs = append(objs, &k.certificate)

	return objs
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

	// TODO: Maybe add more detailed validation. DeepEquals doesn't work because of added fields from k8s

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
						Name:      "kudo-controller-manager-service",
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
				"commonName": "kudo-controller-manager-service.kudo-system.svc",
				"dnsNames":   []string{"kudo-controller-manager-service.kudo-system.svc"},
				"issuerRef": map[string]interface{}{
					"kind": "Issuer",
					"name": "selfsigned-issuer",
				},
				"secretName": "kudo-webhook-server-secret",
			},
		},
	}
}
