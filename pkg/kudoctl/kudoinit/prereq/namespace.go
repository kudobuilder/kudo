package prereq

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"

	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Ensure IF is implemented
var _ k8sResource = &kudoNamespace{}

type kudoNamespace struct {
	opts kudoinit.Options
	ns   *v1.Namespace
}

func (o kudoNamespace) PreInstallCheck(client *kube.Client) error {
	// We only manage kudo-system namespace. For others we expect they exist.
	if !o.opts.IsDefaultNamespace() {
		_, err := client.KubeClient.CoreV1().Namespaces().Get(o.opts.Namespace, metav1.GetOptions{})
		if err == nil {
			return nil
		}
		if kerrors.IsNotFound(err) {
			return fmt.Errorf("namespace %s does not exist - KUDO expects that any namespace except the default %s is created beforehand", o.opts.Namespace, kudoinit.DefaultNamespace)
		}
		return err
	}
	return nil
}

func newNamespace(options kudoinit.Options) kudoNamespace {
	return kudoNamespace{
		opts: options,
		ns:   generateSysNamespace(options.Namespace),
	}
}

func (o kudoNamespace) Install(client *kube.Client) error {
	_, err := client.KubeClient.CoreV1().Namespaces().Create(o.ns)
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("namespace %v already exists", o.ns.Name)
		return nil
	}
	return err
}

func (o kudoNamespace) ValidateInstallation(client *kube.Client) error {
	return nil
}

func (o kudoNamespace) AsRuntimeObjs() []runtime.Object {
	if !o.opts.IsDefaultNamespace() {
		return make([]runtime.Object, 0)
	}
	return []runtime.Object{o.ns}
}

// generateSysNamespace builds the system namespace
func generateSysNamespace(namespace string) *v1.Namespace {
	labels := kudoinit.GenerateLabels(map[string]string{"controller-tools.k8s.io": "1.0"})
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
