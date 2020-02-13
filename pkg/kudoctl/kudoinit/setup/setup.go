package setup

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/crd"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/manager"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/prereq"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verify"
)

// Validate checks that the current KUDO installation is correct
func Validate(client *kube.Client, opts kudoinit.Options) error {
	initSteps := initSteps(opts, false)

	result := verify.NewResult()
	// Check if all steps are correctly installed
	for _, initStep := range initSteps {
		result.Merge(initStep.VerifyInstallation(client))
	}

	return nil
}

// Upgrade an existing KUDO installation
func Upgrade(client *kube.Client, opts kudoinit.Options) error {
	initSteps := initSteps(opts, false)

	// Step 1 - Verify that installation can be done
	// Check if all steps are installable
	result := verify.NewResult()
	for _, initStep := range initSteps {
		result.Merge(initStep.PreInstallVerify(client))
	}
	if !result.IsValid() {
		return &result
	}

	// Step 2 - Verify that any migration is possible
	// TODO: Determine which migrations to run and execute PreInstallVerify

	// Step 3 - Shut down/remove manager
	// Step 4 - Disable Admission-Webhooks

	// Step 5 - Execute Migrations

	// Step 6 - Execute Installation/Upgrade (this enables webhooks again and starts new manager

	return nil
}

// Install uses Kubernetes client to install KUDO.
func Install(client *kube.Client, opts kudoinit.Options, crdOnly bool) error {
	initSteps := initSteps(opts, crdOnly)

	result := verify.NewResult()
	// Check if all steps are installable
	for _, initStep := range initSteps {
		result.Merge(initStep.PreInstallVerify(client))
	}

	result.PrintWarnings(os.Stdout)
	if !result.IsValid() {
		return &result
	}

	// Install everything
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
