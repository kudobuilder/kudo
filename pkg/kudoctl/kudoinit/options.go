package kudoinit

import (
	"fmt"

	v1 "k8s.io/api/core/v1"

	"github.com/kudobuilder/kudo/pkg/version"
)

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
	// Image PullPolicy
	PullPolicy v1.PullPolicy

	// Enable validation
	Webhooks       []string
	ServiceAccount string
	Upgrade        bool
}

func NewOptions(v string, pullPolicy v1.PullPolicy, ns string, sa string, webhooks []string, upgrade bool) Options {
	if pullPolicy == "" {
		pullPolicy = v1.PullAlways
	}
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
		PullPolicy:                    pullPolicy,
		Webhooks:                      webhooks,
		ServiceAccount:                sa,
		Upgrade:                       upgrade,
	}
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
