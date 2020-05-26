package prereq

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
		"failed to find CRD 'certificates.cert-manager.io': customresourcedefinitions.apiextensions.k8s.io \"certificates.cert-manager.io\" not found",
		"failed to find CRD 'issuers.cert-manager.io': customresourcedefinitions.apiextensions.k8s.io \"issuers.cert-manager.io\" not found",
	), result)
}

func TestPrereq_Fail_PreValidate_Webhook_WrongCertificateVersion(t *testing.T) {
	client := getFakeClient()

	mockCRD(client, "certificates.cert-manager.io", "v0")
	mockCRD(client, "issuers.cert-manager.io", "v0")

	init := NewWebHookInitializer(kudoinit.NewOptions("", "", "", false, false))

	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.EqualValues(t, verifier.NewError(
		"invalid CRD version found for 'certificates.cert-manager.io': v0 instead of v1alpha2",
		"invalid CRD version found for 'issuers.cert-manager.io': v0 instead of v1alpha2",
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

	assert.EqualValues(t, verifier.NewError("invalid CRD version found for 'issuers.cert-manager.io': v0 instead of v1alpha2"), result)
}

func TestPrereq_Ok_PreValidate_Webhook(t *testing.T) {
	client := getFakeClient()

	mockCRD(client, "certificates.cert-manager.io", "v1alpha2")
	mockCRD(client, "issuers.cert-manager.io", "v1alpha2")

	init := NewWebHookInitializer(kudoinit.NewOptions("", "", "", false, false))

	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.Equal(t, 0, len(result.Errors))
}

func mockCRD(client *kube.Client, crdName string, apiVersion string) {
	client.ExtClient.(*apiextensionsfake.Clientset).Fake.PrependReactor("get", "customresourcedefinitions", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {

		getAction, _ := action.(testing2.GetAction)
		if getAction != nil {
			if getAction.GetName() == crdName {
				crd := &apiextensions.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						APIVersion: apiVersion,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: crdName,
					},
					Spec: apiextensions.CustomResourceDefinitionSpec{
						Versions: []apiextensions.CustomResourceDefinitionVersion{
							{
								Name: apiVersion,
							},
						},
					},
					Status: apiextensions.CustomResourceDefinitionStatus{},
				}
				return true, crd, nil
			}
		}

		return false, nil, nil
	})
}
