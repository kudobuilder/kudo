package prereq

import (
	"testing"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	testing2 "k8s.io/client-go/testing"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
)

func TestPrereq_Ok_PreValidate_Webhook_None(t *testing.T) {
	client := getFakeClient()

	init := NewInitializer(kudoinit.NewOptions("", "", "", []string{}))

	result := init.PreInstallCheck(client)

	assert.EqualValues(t, kudoinit.NewResult(), result)
}

func TestPrereq_Fail_PreValidate_Webhook_NoCertificate(t *testing.T) {
	client := getFakeClient()

	init := NewInitializer(kudoinit.NewOptions("", "", "", []string{"validate"}))

	result := init.PreInstallCheck(client)

	assert.EqualValues(t, kudoinit.NewError("cert-manager is not installed, failed to find Certificate CRD: customresourcedefinitions.apiextensions.k8s.io \"Certificate\" not found"), result)
}

func TestPrereq_Fail_PreValidate_Webhook_WrongCertificateVersion(t *testing.T) {
	client := getFakeClient()

	mockCRD(client, "Certificate", "cert-manager.io/v0")

	init := NewInitializer(kudoinit.NewOptions("", "", "", []string{"validate"}))

	result := init.PreInstallCheck(client)

	assert.EqualValues(t, kudoinit.NewError("invalid cert-manager API version found for Certificate CRD: cert-manager.io/v0 instead of cert-manager.io/v1alpha2"), result)
}

func TestPrereq_Fail_PreValidate_Webhook_NoIssuer(t *testing.T) {
	client := getFakeClient()

	mockCRD(client, "Certificate", "cert-manager.io/v1alpha2")

	init := NewInitializer(kudoinit.NewOptions("", "", "", []string{"validate"}))

	result := init.PreInstallCheck(client)

	assert.EqualValues(t, kudoinit.NewError("cert-manager is not installed, failed to find Issuer CRD: customresourcedefinitions.apiextensions.k8s.io \"Issuer\" not found"), result)
}

func TestPrereq_Fail_PreValidate_Webhook_WrongIssuerVersion(t *testing.T) {
	client := getFakeClient()

	mockCRD(client, "Certificate", "cert-manager.io/v1alpha2")
	mockCRD(client, "Issuer", "cert-manager.io/v0")

	init := NewInitializer(kudoinit.NewOptions("", "", "", []string{"validate"}))

	result := init.PreInstallCheck(client)

	assert.EqualValues(t, kudoinit.NewError("invalid cert-manager API version found for Issuer CRD: cert-manager.io/v0 instead of cert-manager.io/v1alpha2"), result)
}

func TestPrereq_Ok_PreValidate_Webhook(t *testing.T) {
	client := getFakeClient()

	mockCRD(client, "Certificate", "cert-manager.io/v1alpha2")
	mockCRD(client, "Issuer", "cert-manager.io/v1alpha2")

	init := NewInitializer(kudoinit.NewOptions("", "", "", []string{"validate"}))

	result := init.PreInstallCheck(client)

	assert.EqualValues(t, kudoinit.NewResult(), result)
}

func mockCRD(client *kube.Client, crdName string, apiVersion string) {
	client.ExtClient.(*apiextensionsfake.Clientset).Fake.PrependReactor("get", "customresourcedefinitions", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {

		getAction, _ := action.(testing2.GetAction)
		if getAction != nil {
			if getAction.GetName() == crdName {
				crd := &apiextensions.CustomResourceDefinition{
					TypeMeta: v12.TypeMeta{
						APIVersion: apiVersion,
					},
					ObjectMeta: v12.ObjectMeta{
						Name: crdName,
					},
					Spec:   apiextensions.CustomResourceDefinitionSpec{},
					Status: apiextensions.CustomResourceDefinitionStatus{},
				}
				return true, crd, nil
			}
		}

		return false, nil, nil
	})
}
