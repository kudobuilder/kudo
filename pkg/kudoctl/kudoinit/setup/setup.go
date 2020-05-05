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

// Verifies that the installation is possible. Returns an error if any part of KUDO is already installed
func PreInstallVerify(client *kube.Client, opts kudoinit.Options, crdOnly bool, result *verifier.Result) error {
	initSteps := initSteps(opts, crdOnly)

	// Check if all steps are installable
	for _, initStep := range initSteps {
		if err := initStep.PreInstallVerify(client, result); err != nil {
			return fmt.Errorf("error while verifying install step %s: %v", initStep.String(), err)
		}
	}

	return nil
}

// Install uses Kubernetes client to install KUDO.
func Install(client *kube.Client, opts kudoinit.Options, crdOnly bool) error {
	// Install everything
	initSteps := initSteps(opts, crdOnly)
	for _, initStep := range initSteps {
		if err := initStep.Install(client); err != nil {
			return fmt.Errorf("%s: %v", initStep, err)
		}
		clog.Printf("✅ installed %s", initStep)
	}

	return nil
}

func initSteps(opts kudoinit.Options, crdOnly bool) []kudoinit.Step {
	if crdOnly {
		return []kudoinit.Step{
			crd.NewInitializer(),
		}
	}

	return []kudoinit.Step{
		crd.NewInitializer(),
		prereq.NewNamespaceInitializer(opts),
		prereq.NewServiceAccountInitializer(opts),
		prereq.NewWebHookInitializer(opts),
		manager.NewInitializer(opts),
	}
}

func AsYamlManifests(opts kudoinit.Options, crdOnly bool) ([]string, error) {
	initSteps := initSteps(opts, crdOnly)
	var allManifests []runtime.Object

	for _, initStep := range initSteps {
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
