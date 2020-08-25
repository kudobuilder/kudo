package prereq

import (
	"testing"

	"github.com/stretchr/testify/assert"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	testing2 "k8s.io/client-go/testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

func TestPrereq_Fail_PreValidate_CustomNamespace(t *testing.T) {
	client := getFakeClient()

	init := NewNamespaceInitializer(kudoinit.NewOptions("", "customNS", "", true, true))
	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.EqualValues(t, verifier.NewError("Namespace customNS does not exist - KUDO expects that any namespace except the default kudo-system is created beforehand"), result)
}

func TestPrereq_Ok_PreValidate_CustomNamespace(t *testing.T) {
	client := getFakeClient()

	mockGetNamespace(client, "customNS", false)

	init := NewNamespaceInitializer(kudoinit.NewOptions("", "customNS", "", true, true))

	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.EqualValues(t, verifier.NewResult(), result)
}

func TestPrereq_Fail_DefaultNamespaceTerminating(t *testing.T) {
	client := getFakeClient()

	mockGetNamespace(client, "kudo-system", true)

	init := NewNamespaceInitializer(kudoinit.NewOptions("", "", "", true, true))

	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	expectedResult := verifier.NewResult()
	expectedResult.AddErrors("Namespace kudo-system is being terminated - Wait until it is fully gone and retry")

	assert.EqualValues(t, expectedResult, result)
}

func mockGetNamespace(client *kube.Client, nsName string, terminating bool) {
	client.KubeClient.(*fake.Clientset).Fake.PrependReactor("get", "namespaces", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
		ns := &core.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: nsName,
			},
		}
		if terminating {
			ns.Status = core.NamespaceStatus{
				Phase: core.NamespaceTerminating,
			}
		}
		return true, ns, nil
	})
}
