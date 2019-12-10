package setup

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/version"
)

const (
	defaultNamespace      = "kudo-system"
	defaultGracePeriod    = 10
	defaultServiceAccount = "kudo-manager"
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
	// Enable validation
	Webhooks       []string
	ServiceAccount string
}

func (o Options) hasWebhooksEnabled() bool {
	return len(o.Webhooks) != 0
}

func (o Options) isDefaultNamespace() bool {
	return o.Namespace == defaultNamespace
}

func NewOptions(v string, ns string, sa string, webhooks []string) Options {
	if v == "" {
		v = version.Get().GitVersion
	}
	if ns == "" {
		ns = defaultNamespace
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

// Install uses Kubernetes client to install KUDO.
func Install(client *kube.Client, opts Options, crdOnly bool) error {

	clog.Printf("✅ installing crds")
	if err := CRDs().Install(client); err != nil {
		return err
	}
	if crdOnly {
		return nil
	}

	clog.Printf("✅ preparing service accounts and other requirements for controller to run")
	if err := Prereqs(opts).Install(client); err != nil {
		return err
	}

	clog.Printf("✅ installing kudo controller")
	if err := Manager(opts).Install(client); err != nil {
		return err
	}
	return nil
}

func AsYamlManifests(opts Options, crdOnly bool) ([]string, error) {
	var manifests []string

	crd, err := CRDs().AsYamlManifests()
	if err != nil {
		return nil, err
	}
	manifests = append(manifests, crd...)

	if crdOnly {
		return manifests, nil
	}

	prereqs, err := Prereqs(opts).AsYamlManifests()
	if err != nil {
		return nil, err
	}
	manifests = append(manifests, prereqs...)

	mgr, err := Manager(opts).AsYamlManifests()
	if err != nil {
		return nil, err
	}
	manifests = append(manifests, mgr...)

	return manifests, err
}

func generateLabels(labels map[string]string) map[string]string {
	labels["app"] = "kudo-manager"
	return labels
}
