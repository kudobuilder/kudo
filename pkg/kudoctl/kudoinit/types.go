package kudoinit

import (
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
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

	// InstalledVersion returns the currently installed version of this step, or nil if it is not installed
	InstalledVersion(client *kube.Client) (string, error)
}

func GenerateLabels(labels map[string]string) map[string]string {
	labels["app"] = "kudo-manager"
	return labels
}
