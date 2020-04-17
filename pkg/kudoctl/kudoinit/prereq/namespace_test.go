package prereq

import (
	"testing"

	core "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	testing2 "k8s.io/client-go/testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

func TestPrereq_Fail_PreValidate_CustomNamespace(t *testing.T) {
	client := getFakeClient()

	init := NewNamespaceInitializer(kudoinit.NewOptions("", "", "customNS", "", make([]string, 0), false, false))
	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.EqualValues(t, verifier.NewError("Namespace customNS does not exist - KUDO expects that any namespace except the default kudo-system is created beforehand"), result)
}

func TestPrereq_Ok_PreValidate_CustomNamespace(t *testing.T) {
	client := getFakeClient()

	mockGetNamespace(client, "customNS")

	init := NewNamespaceInitializer(kudoinit.NewOptions("", "", "customNS", "", make([]string, 0), false, false))

	result := verifier.NewResult()
	_ = init.PreInstallVerify(client, &result)

	assert.EqualValues(t, verifier.NewResult(), result)
}

func mockGetNamespace(client *kube.Client, nsName string) {
	client.KubeClient.(*fake.Clientset).Fake.PrependReactor("get", "namespaces", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
		ns := &core.Namespace{
			ObjectMeta: v12.ObjectMeta{
				Name: nsName,
			},
		}
		return true, ns, nil
	})
}
