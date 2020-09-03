package kudoinit

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

const (
	DefaultManagerName    = "kudo-controller-manager"
	DefaultNamespace      = "kudo-system"
	DefaultServiceName    = "kudo-controller-manager-service"
	DefaultSecretName     = "kudo-webhook-server-secret" //nolint
	DefaultKudoLabel      = "kudo-manager"
	ManagerContainerName  = "manager"
	defaultGracePeriod    = 10
	defaultServiceAccount = "kudo-manager"
)

type Artifacter interface {
	// Returns the artifacts that would be installed as runtime objects
	Resources() []runtime.Object
}

type InstallVerifier interface {
	// PreInstallVerify verifies that the installation is possible
	PreInstallVerify(client *kube.Client, result *verifier.Result) error

	// PreInstallVerify verifies that an upgrade to the new version is possible
	PreUpgradeVerify(client *kube.Client, result *verifier.Result) error

	// VerifyInstallation verifies that the current installation is as expected
	VerifyInstallation(client *kube.Client, result *verifier.Result) error
}

type Installer interface {
	// Executes the actual installation
	Install(client *kube.Client) error
}

type Step interface {
	fmt.Stringer

	InstallVerifier
	Installer

	Artifacter
}

func GenerateLabels(labels map[string]string) map[string]string {
	labels["app"] = DefaultKudoLabel
	return labels
}
