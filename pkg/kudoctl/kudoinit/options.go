package kudoinit

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/version"
)

// Options is the configurable options to init
type Options struct {
	// Version is the version of the manager `0.5.0` for example (must NOT include the `v` in `v0.5.0`)
	Version string
	// Name is used as controller-manager Name and as a prefix for the service. Currently hard-coded
	// to "kudo-controller-manager" as we don't support setting it in the CLI.
	Name string
	// namespace to init into (default is kudo-system)
	Namespace string
	// TerminationGracePeriodSeconds defines the termination grace period for a pod
	TerminationGracePeriodSeconds int64
	// Image defines the image to be used
	Image string
	// List of enabled webhooks
	Webhooks []string
	// Using self-signed webhook CA bundle
	SelfSignedWebhookCA bool
	// ServiceName is the controllers service name. Currently hard-coded to "kudo-controller-manager-service"
	// as we don't support setting it in the CLI
	ServiceName string
	// SecretName is used by the manager when the webhooks are activated. Contains certificate and the key files for
	// the webhook server. Currently hard-coded to "kudo-webhook-server-secret" as we don't support setting it in the CLI
	SecretName string

	ServiceAccount string
}

func NewOptions(v string, ns string, sa string, webhooks []string, selfSignedWebhookCA bool) Options {
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
		Name:                          DefaultName,
		ServiceName:                   DefaultServiceName,
		SecretName:                    DefaultSecretName,
		Namespace:                     ns,
		TerminationGracePeriodSeconds: defaultGracePeriod,
		Image:                         fmt.Sprintf("kudobuilder/controller:v%v", v),
		Webhooks:                      webhooks,
		ServiceAccount:                sa,
		SelfSignedWebhookCA:           selfSignedWebhookCA,
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
