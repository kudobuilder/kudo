package kudoinit

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
)

const (
	DefaultNamespace      = "kudo-system"
	defaultGracePeriod    = 10
	defaultServiceAccount = "kudo-manager"
)

type StepArtifacts interface {
	// Returns the installed artifacts as yaml manifests
	AsYamlManifests() ([]string, error)
	AsArray() []runtime.Object
}

type StepInstallation interface {
	// Should return an error if the installation will not be possible
	PreInstallCheck(client *kube.Client) Result

	// Executes the actual installation
	Install(client *kube.Client) error
}

type Step interface {
	fmt.Stringer

	StepInstallation
	StepArtifacts
}

func GenerateLabels(labels map[string]string) map[string]string {
	labels["app"] = "kudo-manager"
	return labels
}
