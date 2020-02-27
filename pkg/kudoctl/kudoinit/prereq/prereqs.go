package prereq

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

// Ensure kudoinit.Step is implemented
var _ kudoinit.Step = &Initializer{}

// Defines a single prerequisite that is defined as a k8s resource
type k8sResource interface {
	// PreInstallVerify is called before the installation of any component is started and should return an error if the installation is not possible
	PreInstallVerify(client *kube.Client) verifier.Result

	// Install installs the manifests of this prerequisite
	Install(client *kube.Client) error

	// ValidateInstallation verifies that the current state of the installation is as expected of this version of KUDO
	ValidateInstallation(client *kube.Client) error

	// AsRuntimeObjs returns the manifests that would be installed from this resource
	AsRuntimeObjs() []runtime.Object
}

//Defines the Prerequisites that need to be in place to run the KUDO manager. This includes setting up the kudo-system namespace and service account
type Initializer struct {
	Options kudoinit.Options
	prereqs []k8sResource
}

func NewInitializer(options kudoinit.Options) Initializer {
	return Initializer{
		Options: options,
		prereqs: []k8sResource{
			newNamespace(options),
			newServiceAccount(options),
			newWebHook(options),
		},
	}
}

func (p Initializer) String() string {
	return "service accounts and other requirements for controller to run"
}

func (p Initializer) PreInstallVerify(client *kube.Client) verifier.Result {
	result := verifier.NewResult()
	for _, prereq := range p.prereqs {
		res := prereq.PreInstallVerify(client)
		result.Merge(res)
	}
	return result
}

func (p Initializer) Install(client *kube.Client) error {
	for _, prereq := range p.prereqs {
		err := prereq.Install(client)
		if err != nil {
			return fmt.Errorf("failed to install: %v", err)
		}
	}
	return nil
}

func (p Initializer) Resources() []runtime.Object {
	var prereqs []runtime.Object

	for _, prereq := range p.prereqs {
		prereqs = append(prereqs, prereq.AsRuntimeObjs()...)
	}
	return prereqs
}
