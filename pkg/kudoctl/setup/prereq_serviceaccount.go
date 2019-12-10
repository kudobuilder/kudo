package setup

import (
	"fmt"
	"reflect"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"

	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type KudoServiceAccount struct {
	opts           Options
	serviceAccount *v1.ServiceAccount
}

func newServiceAccountSetup(options Options) KudoServiceAccount {
	return KudoServiceAccount{
		opts:           options,
		serviceAccount: generateServiceAccount(options),
	}
}

func (o KudoServiceAccount) Install(client *kube.Client) error {
	coreClient := client.KubeClient.CoreV1()
	_, err := coreClient.ServiceAccounts(o.opts.Namespace).Create(o.serviceAccount)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("service account %v already exists", o.serviceAccount.Name)
		return nil
	}
	return err
}

func (o KudoServiceAccount) Validate(client *kube.Client) error {
	coreClient := client.KubeClient.CoreV1()
	existing, err := coreClient.ServiceAccounts(o.opts.Namespace).Get(o.serviceAccount.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to retrieve service account %v", err)
	}

	if !reflect.DeepEqual(existing, o.serviceAccount) {
		return fmt.Errorf("installed ServiceAccount does not equal expected service account")
	}

	return nil
}

func (o KudoServiceAccount) AsRuntimeObj() []runtime.Object {
	return []runtime.Object{o.serviceAccount}
}

// generateServiceAccount builds the system account
func generateServiceAccount(opts Options) *v1.ServiceAccount {
	labels := generateLabels(map[string]string{})
	return &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      opts.ServiceAccount,
			Namespace: opts.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: "v1",
		},
	}
}
