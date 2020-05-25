package prereq

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

// Ensure IF is implemented
var _ kudoinit.Step = &KudoNamespace{}

type KudoNamespace struct {
	opts kudoinit.Options
	ns   *v1.Namespace
}

func NewNamespaceInitializer(options kudoinit.Options) KudoNamespace {
	return KudoNamespace{
		opts: options,
		ns:   generateSysNamespace(options.Namespace),
	}
}

func (o KudoNamespace) String() string {
	return "namespace"
}

func (o KudoNamespace) PreInstallVerify(client *kube.Client, result *verifier.Result) error {
	// We only manage kudo-system namespace. For others we expect they exist.
	if !o.opts.IsDefaultNamespace() {
		_, err := client.KubeClient.CoreV1().Namespaces().Get(context.TODO(), o.opts.Namespace, metav1.GetOptions{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				result.AddErrors(fmt.Sprintf("Namespace %s does not exist - KUDO expects that any namespace except the default %s is created beforehand", o.opts.Namespace, kudoinit.DefaultNamespace))
				return nil
			}
			return err
		}
	}
	return nil
}

func (o KudoNamespace) Install(client *kube.Client) error {
	_, err := client.KubeClient.CoreV1().Namespaces().Create(context.TODO(), o.ns, metav1.CreateOptions{})
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("namespace %v already exists", o.ns.Name)
		return nil
	}
	return err
}

func (o KudoNamespace) Resources() []runtime.Object {
	if !o.opts.IsDefaultNamespace() {
		return make([]runtime.Object, 0)
	}
	return []runtime.Object{o.ns}
}

// generateSysNamespace builds the system namespace
func generateSysNamespace(namespace string) *v1.Namespace {
	labels := kudoinit.GenerateLabels(map[string]string{})
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
			Name:   namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
	}
}
