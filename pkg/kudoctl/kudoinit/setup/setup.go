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

var _ kudoinit.InstallVerifier = &Installer{}

type Installer struct {
	options kudoinit.Options
	steps   []kudoinit.Step

	managerStep *manager.Initializer
	webhookStep *prereq.KudoWebHook
}

func NewInstaller(options kudoinit.Options, crdOnly bool) *Installer {
	if crdOnly {
		return &Installer{
			options: options,
			steps: []kudoinit.Step{
				crd.NewInitializer(),
			},
		}
	}

	// This is a bit cumbersome - we need to access some funcs from these two
	// steps for the upgrade process, that's why they are initialized here.
	// This should be cleaned up
	managerStep := manager.NewInitializer(options)
	webhookStep := prereq.NewWebHookInitializer(options)

	return &Installer{
		options:     options,
		managerStep: managerStep,
		webhookStep: webhookStep,
		steps: []kudoinit.Step{
			crd.NewInitializer(),
			prereq.NewNamespaceInitializer(options),
			prereq.NewServiceAccountInitializer(options),
			webhookStep,
			managerStep,
		},
	}
}

// Validate checks that the current KUDO installation is correct
func (i *Installer) VerifyInstallation(client *kube.Client, result *verifier.Result) error {
	// Check if all steps are correctly installed
	for _, initStep := range i.steps {
		if err := initStep.VerifyInstallation(client, result); err != nil {
			return fmt.Errorf("error while verifying init step %s: %v", initStep.String(), err)
		}
	}

	return nil
}

func requiredMigrations() []migration.Migrater {

	// Determine which migrations to run
	return []migration.Migrater{
		// Implement actual migrations
	}
}

func (i *Installer) PreUpgradeVerify(client *kube.Client, result *verifier.Result) error {
	// Step 1 - Verify that upgrade can be done
	// Check if all steps are upgradeable
	for _, initStep := range i.steps {
		if err := initStep.PreUpgradeVerify(client, result); err != nil {
			return fmt.Errorf("error while verifying upgrade step %s: %v", initStep.String(), err)
		}
	}
	if !result.IsValid() {
		return nil
	}

	// Step 2 - Verify that any migration is possible
	migrations := requiredMigrations()
	clog.Printf("Verify that %d required migrations can be applied", len(migrations))
	for _, m := range migrations {
		if err := m.CanMigrate(client); err != nil {
			result.AddErrors(fmt.Errorf("migration %s failed install check: %v", m, err).Error())
		}
	}

	// TODO: Verify existing operators and instances?

	return nil
}

// Upgrade an existing KUDO installation
func (i *Installer) Upgrade(client *kube.Client) error {
	clog.Printf("Upgrade KUDO")

	// Step 3 - Shut down/remove manager
	if err := i.managerStep.UninstallStatefulSet(client); err != nil {
		return fmt.Errorf("failed to uninstall existing KUDO manager: %v", err)
	}

	// Step 4 - Disable Admission-Webhooks
	if err := i.webhookStep.UninstallWebHook(client); err != nil {
		return fmt.Errorf("failed to uninstall webhook: %v", err)
	}

	// Step 5 - Execute Migrations
	migrations := requiredMigrations()
	clog.Printf("Run %d migrations", len(migrations))
	for _, m := range migrations {
		if err := m.Migrate(client); err != nil {
			return fmt.Errorf("migration %s failed to execute: %v", m, err)
		}
	}

	// Step 6 - Execute Installation/Upgrade (this enables webhooks again and starts new manager
	return i.Install(client)
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
