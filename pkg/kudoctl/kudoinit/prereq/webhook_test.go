package prereq

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	testing2 "k8s.io/client-go/testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

func TestPrereq_Ok_PreValidate_Webhook_None(t *testing.T) {
	client := getFakeClient()

	init := NewWebHookInitializer(kudoinit.NewOptions("", "", "", false, true))

	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.EqualValues(t, verifier.NewResult(), result)
}

func TestPrereq_Fail_PreValidate_Webhook_NoCertificate(t *testing.T) {
	client := getFakeClient()

	init := NewWebHookInitializer(kudoinit.NewOptions("", "", "", false, false))

	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.EqualValues(t, verifier.NewError(
		"failed to detect any valid cert-manager CRDs. Make sure cert-manager is installed.",
	), result)
}

func TestPrereq_Fail_PreValidate_Webhook_WrongCertificateVersion(t *testing.T) {
	client := getFakeClient()

	mockCRD(client, "certificates.cert-manager.io", "v0")
	mockCRD(client, "issuers.cert-manager.io", "v0")

	init := NewWebHookInitializer(kudoinit.NewOptions("", "", "", false, false))

	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.EqualValues(t, verifier.NewWarning(
		"Detected cert-manager CRDs with version v0, only versions [v1 v1beta1 v1alpha3 v1alpha2] are fully supported. Certificates for webhooks may not work.",
	), result)
}

func TestPrereq_Fail_PreValidate_Webhook_WrongCertManagerInstallation(t *testing.T) {
	client := getFakeClient()

	mockCRD(client, "certificates.cert-manager.io", "v1alpha2")
	mockCRD(client, "issuers.cert-manager.io", "v0")

	init := NewWebHookInitializer(kudoinit.NewOptions("", "", "", false, false))

	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.EqualValues(t, verifier.NewError(
		"CRD versions for 'issuers.cert-manager.io' are [v0], did not find expected version v1alpha2 or it is not served",
	), result)
}

func TestPrereq_Fail_PreValidate_Webhook_NoIssuer(t *testing.T) {
	client := getFakeClient()

	mockCRD(client, "certificates.cert-manager.io", "v1alpha2")

	init := NewWebHookInitializer(kudoinit.NewOptions("", "", "", false, false))

	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.EqualValues(t, verifier.NewError("failed to find CRD 'issuers.cert-manager.io': customresourcedefinitions.apiextensions.k8s.io \"issuers.cert-manager.io\" not found"), result)
}

func TestPrereq_Fail_PreValidate_Webhook_WrongIssuerVersion(t *testing.T) {
	client := getFakeClient()

	mockCRD(client, "certificates.cert-manager.io", "v1alpha2")
	mockCRD(client, "issuers.cert-manager.io", "v0")

	init := NewWebHookInitializer(kudoinit.NewOptions("", "", "", false, false))

	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.EqualValues(t, verifier.NewError("CRD versions for 'issuers.cert-manager.io' are [v0], did not find expected version v1alpha2 or it is not served"), result)
}

func TestPrereq_Ok_PreValidate_Webhook_CertManager_v1alpha2(t *testing.T) {
	client := getFakeClient()

	mockCRD(client, "certificates.cert-manager.io", "v1alpha2")
	mockCRD(client, "issuers.cert-manager.io", "v1alpha2")

	init := NewWebHookInitializer(kudoinit.NewOptions("", "", "", false, false))

	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.Equal(t, 0, len(result.Errors))
}

func TestPrereq_Ok_PreValidate_Webhook_CertManager_v1alpha1(t *testing.T) {
	client := getFakeClient()

	mockCRD(client, "certificates.certmanager.k8s.io", "v1alpha1")
	mockCRD(client, "issuers.certmanager.k8s.io", "v1alpha1")

	init := NewWebHookInitializer(kudoinit.NewOptions("", "", "", false, false))

	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.Equal(t, 0, len(result.Errors))
}

func TestPrereq_Ok_PreValidate_Webhook_CertManager_MultipleVersions(t *testing.T) {
	client := getFakeClient()

	cert := createCrd("certificates.certmanager.k8s.io", "v1alpha1")
	cert.Spec.Versions = append(cert.Spec.Versions, apiextv1.CustomResourceDefinitionVersion{
		Name:    "v1beta1",
		Served:  true,
		Storage: false,
	})
	cert.Spec.Versions = append(cert.Spec.Versions, apiextv1.CustomResourceDefinitionVersion{
		Name:    "v1beta2",
		Served:  true,
		Storage: false,
	})

	issuer := createCrd("issuers.certmanager.k8s.io", "v1alpha1")
	issuer.Spec.Versions = append(cert.Spec.Versions, apiextv1.CustomResourceDefinitionVersion{ //nolint:gocritic
		Name:    "v1beta1",
		Served:  true,
		Storage: false,
	})
	issuer.Spec.Versions = append(cert.Spec.Versions, apiextv1.CustomResourceDefinitionVersion{ //nolint:gocritic
		Name:    "v1beta2",
		Served:  true,
		Storage: false,
	})

	mockFullCRD(client, cert)
	mockFullCRD(client, issuer)

	init := NewWebHookInitializer(kudoinit.NewOptions("", "", "", false, false))

	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.Equal(t, init.certManagerAPIVersion, "v1beta2")

	assert.Equal(t, 0, len(result.Errors))
}

func mockFullCRD(client *kube.Client, definition *apiextv1.CustomResourceDefinition) {
	client.ExtClient.(*apiextensionsfake.Clientset).Fake.PrependReactor("get", "customresourcedefinitions", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
		getAction, _ := action.(testing2.GetAction)
		if getAction != nil {
			if getAction.GetName() == definition.Name {
				return true, definition, nil
			}
		}
		return false, nil, nil
	})
}

func createCrd(crdName string, apiVersion string) *apiextv1.CustomResourceDefinition {
	return &apiextv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: crdName,
		},
		Spec: apiextv1.CustomResourceDefinitionSpec{
			Versions: []apiextv1.CustomResourceDefinitionVersion{
				{
					Name:    apiVersion,
					Served:  true,
					Storage: true,
				},
			},
		},
		Status: apiextv1.CustomResourceDefinitionStatus{},
	}
}

func mockCRD(client *kube.Client, crdName string, apiVersion string) {
	def := createCrd(crdName, apiVersion)
	mockFullCRD(client, def)

}
