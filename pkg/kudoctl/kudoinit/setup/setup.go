package setup

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/crd"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/manager"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/prereq"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

type Installer struct {
	steps []kudoinit.Step
}

func NewInstaller(options kudoinit.Options, crdOnly bool) *Installer {
	if crdOnly {
		return &Installer{
			steps: []kudoinit.Step{
				crd.NewInitializer(),
			},
		}
	}

	return &Installer{
		steps: []kudoinit.Step{
			crd.NewInitializer(),
			prereq.NewNamespaceInitializer(options),
			prereq.NewServiceAccountInitializer(options),
			prereq.NewWebHookInitializer(options),
			manager.NewInitializer(options),
		},
	}
}

// Verifies that the installation is possible. Returns an error if any part of KUDO is already installed
func (i *Installer) PreInstallVerify(client *kube.Client, result *verifier.Result) error {
	if _, err := client.KubeClient.Discovery().ServerVersion(); err != nil {
		result.AddErrors(fmt.Sprintf("Failed to connect to cluster: %v", err))
		return nil
	}

	// Check if all steps are installable
	for _, initStep := range i.steps {
		if err := initStep.PreInstallVerify(client, result); err != nil {
			return fmt.Errorf("error while verifying install step %s: %v", initStep.String(), err)
		}
	}

	return nil
}

// Install uses Kubernetes client to install KUDO.
func (i *Installer) Install(client *kube.Client) error {
	// Install everything
	initSteps := i.steps
	for _, initStep := range initSteps {
		if err := initStep.Install(client); err != nil {
			return fmt.Errorf("%s: %v", initStep, err)
		}
		clog.Printf("✅ installed %s", initStep)
	}

	return nil
}

func (i *Installer) AsYamlManifests() ([]string, error) {
	var allManifests []runtime.Object

	for _, initStep := range i.steps {
		allManifests = append(allManifests, initStep.Resources()...)
	}

	return toYaml(allManifests)
}

func toYaml(objs []runtime.Object) ([]string, error) {
	manifests := make([]string, len(objs))
	for i, obj := range objs {
		o, err := yaml.Marshal(obj)
		if err != nil {
			return []string{}, err
		}
		manifests[i] = string(o)
	}

	return manifests, nil
}
