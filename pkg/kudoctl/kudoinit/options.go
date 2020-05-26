package kudoinit

import (
	"fmt"

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
	// ImagePullPolicy sets the pull policy of the image
	ImagePullPolicy string
	// Using self-signed webhook CA bundle
	SelfSignedWebhookCA bool

	ServiceAccount string
	Upgrade        bool
}

func NewOptions(v string, ns string, sa string, upgrade bool, selfSignedWebhookCA bool) Options {
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
		ImagePullPolicy:               "Always",
		ServiceAccount:                sa,
		Upgrade:                       upgrade,
		SelfSignedWebhookCA:           selfSignedWebhookCA,
	}
}

func (o Options) IsDefaultNamespace() bool {
	return o.Namespace == DefaultNamespace
}

func (o Options) IsDefaultServiceAccount() bool {
	return o.ServiceAccount == defaultServiceAccount
}
