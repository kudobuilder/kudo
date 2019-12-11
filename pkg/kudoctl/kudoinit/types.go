package kudoinit

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"

	"github.com/kudobuilder/kudo/pkg/version"
)

const (
	DefaultNamespace      = "kudo-system"
	defaultGracePeriod    = 10
	defaultServiceAccount = "kudo-manager"
)

type InitStep interface {
	Install(client *kube.Client) error
	AsYamlManifests() ([]string, error)
	AsArray() []runtime.Object
}

// Options is the configurable options to init
type Options struct {
	// Version is the version of the manager `0.5.0` for example (must NOT include the `v` in `v0.5.0`)
	Version string
	// namespace to init into (default is kudo-system)
	Namespace string
	// TerminationGracePeriodSeconds defines the termination grace period for a pod
	TerminationGracePeriodSeconds int64
	// Image defines the image to be used
	Image string
	// Enable validation
	Webhooks       []string
	ServiceAccount string
}

func (o Options) HasWebhooksEnabled() bool {
	return len(o.Webhooks) != 0
}

func (o Options) IsDefaultNamespace() bool {
	return o.Namespace == DefaultNamespace
}

func (o Options) IsDefaultServiceAccount() bool {
	return o.ServiceAccount == defaultServiceAccount
}

func NewOptions(v string, ns string, sa string, webhooks []string) Options {
	if v == "" {
		v = version.Get().GitVersion
	}
	if ns == "" {
		ns = DefaultNamespace
	}
	if sa == "" {
		sa = defaultServiceAccount
	}

	return Options{
		Version:                       v,
		Namespace:                     ns,
		TerminationGracePeriodSeconds: defaultGracePeriod,
		Image:                         fmt.Sprintf("kudobuilder/controller:v%v", v),
		Webhooks:                      webhooks,
		ServiceAccount:                sa,
	}
}

func GenerateLabels(labels map[string]string) map[string]string {
	labels["app"] = "kudo-manager"
	return labels
}
