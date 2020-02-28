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
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/migration"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/prereq"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

// Validate checks that the current KUDO installation is correct
func Validate(client *kube.Client, opts kudoinit.Options, result *verifier.Result) error {
	initSteps := initSteps(opts, false)

	// Check if all steps are correctly installed
	for _, initStep := range initSteps {
		if err := initStep.VerifyInstallation(client, result); err != nil {
			return fmt.Errorf("error while verifying init step %s: %v", initStep.String(), err)
		}
	}

	return nil
}

func requiredMigrations() []migration.Migrater {

	// TODO: Determine which migrations to run
	return []migration.Migrater{
		// TODO: Implement actual migrations
	}
}

func PreUpgradeVerify(client *kube.Client, opts kudoinit.Options, result *verifier.Result) error {
	initSteps := initSteps(opts, false)

	// Step 1 - Verify that installation can be done
	// Check if all steps are installable
	for _, initStep := range initSteps {
		if err := initStep.PreInstallVerify(client, result); err != nil {
			return fmt.Errorf("error while verifying upgrade step %s: %v", initStep.String(), err)
		}
	}
	if !result.IsValid() {
		return nil
	}

	// Step 2 - Verify that any migration is possible
	clog.Printf("Check if migrations can be applied")
	migrations := requiredMigrations()
	for _, m := range migrations {
		if err := m.CanMigrate(client); err != nil {
			result.AddErrors(fmt.Errorf("migration %s failed install check: %v", m, err).Error())
		}
	}

	// TODO: Verify existing operators and instances?

	return nil
}

// Upgrade an existing KUDO installation
func Upgrade(client *kube.Client, opts kudoinit.Options) error {
	// Step 3 - Shut down/remove manager
	if err := manager.UninstallStatefulSet(client, opts); err != nil {
		return fmt.Errorf("failed to uninstall existing KUDO manager: %v", err)
	}

	// Step 4 - Disable Admission-Webhooks
	if err := prereq.UninstallWebHook(client, opts); err != nil {
		return fmt.Errorf("failed to uninstall webhook: %v", err)
	}

	// Step 5 - Execute Migrations
	migrations := requiredMigrations()
	for _, m := range migrations {
		if err := m.Migrate(client); err != nil {
			return fmt.Errorf("migration %s failed to execute: %v", m, err)
		}
	}

	// Step 6 - Execute Installation/Upgrade (this enables webhooks again and starts new manager
	return Install(client, opts, false)
}

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
