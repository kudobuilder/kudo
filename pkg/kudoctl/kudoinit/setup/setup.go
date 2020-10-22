package setup

import (
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/runtime"

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
	options      kudoinit.Options
	initializers []kudoinit.Step

	managerInitializer *manager.Initializer
	webhookInitializer *prereq.KudoWebHook
}

func NewInstaller(options kudoinit.Options, crdOnly bool) *Installer {
	if crdOnly {
		return &Installer{
			options: options,
			initializers: []kudoinit.Step{
				crd.NewInitializer(),
			},
		}
	}

	// Having two of the initializers as separate fields here is not the best solution, but
	// we need to access some funcs from these two initializers for the upgrade process,
	// that's why they are initialized here.
	managerStep := manager.NewInitializer(options)
	webhookStep := prereq.NewWebHookInitializer(options)

	return &Installer{
		options:            options,
		managerInitializer: managerStep,
		webhookInitializer: webhookStep,
		initializers: []kudoinit.Step{
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
	// Check if all initializers are correctly installed
	for _, initStep := range i.initializers {
		if err := initStep.VerifyInstallation(client, result); err != nil {
			return fmt.Errorf("error while verifying init step %s: %v", initStep.String(), err)
		}
	}

	return nil
}

func requiredMigrations(client *kube.Client, dryRun bool) []migration.Migrater {
	// Determine which migrations to run
	return []migration.Migrater{}
}

func (i *Installer) PreUpgradeVerify(client *kube.Client, result *verifier.Result) error {
	// Step 1 - Verify that upgrade can be done
	// Check if all initializers are upgradeable
	for _, initStep := range i.initializers {
		if err := initStep.PreUpgradeVerify(client, result); err != nil {
			return fmt.Errorf("error while verifying upgrade step %s: %v", initStep.String(), err)
		}
	}
	if !result.IsValid() {
		return nil
	}

	// Step 2 - Verify that all migrations can run (with dryRun)
	migrations := requiredMigrations(client, true)
	clog.V(1).Printf("Verify that %d required migrations can be applied", len(migrations))
	for _, m := range migrations {
		if err := m.CanMigrate(); err != nil {
			result.AddErrors(fmt.Errorf("migration %s failed install check: %v", m, err).Error())
		}
		if err := m.Migrate(); err != nil {
			result.AddErrors(fmt.Errorf("migration %s failed dry-run: %v", m, err).Error())
		}
	}

	// TODO: Verify existing operators and instances?

	return nil
}

// Upgrade an existing KUDO installation
func (i *Installer) Upgrade(client *kube.Client) error {
	clog.Printf("Upgrade KUDO")

	// Step 3 - Shut down/remove manager
	if err := i.managerInitializer.UninstallStatefulSet(client); err != nil {
		return fmt.Errorf("failed to uninstall existing KUDO manager: %v", err)
	}

	// Step 4 - Disable Admission-Webhooks
	if err := i.webhookInitializer.UninstallWebHook(client); err != nil {
		return fmt.Errorf("failed to uninstall webhook: %v", err)
	}

	// Step 5 - Execute Migrations
	migrations := requiredMigrations(client, false)
	clog.Printf("Run %d migrations", len(migrations))
	for _, m := range migrations {
		if err := m.Migrate(); err != nil {
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

	// Check if all initializers are installable
	for _, initStep := range i.initializers {
		if err := initStep.PreInstallVerify(client, result); err != nil {
			return fmt.Errorf("error while verifying install step %s: %v", initStep.String(), err)
		}
	}

	return nil
}

// Install uses Kubernetes client to install KUDO.
func (i *Installer) Install(client *kube.Client) error {
	// Install everything
	initSteps := i.initializers
	for _, initStep := range initSteps {
		if err := initStep.Install(client); err != nil {
			return fmt.Errorf("%s: %v", initStep, err)
		}
		clog.Printf("✅ installed %s", initStep)
	}
	return nil
}

func (i *Installer) Resources() []runtime.Object {
	var allManifests []runtime.Object

	for _, initStep := range i.initializers {
		allManifests = append(allManifests, initStep.Resources()...)
	}

	return allManifests
}

// verifyExistingInstallation checks if the current installation is valid and as expected
func VerifyExistingInstallation(v kudoinit.InstallVerifier, client *kube.Client, out io.Writer) (bool, error) {
	clog.V(4).Printf("verify existing installation")
	result := verifier.NewResult()
	if err := v.VerifyInstallation(client, &result); err != nil {
		return false, err
	}
	if out != nil {
		result.PrintWarnings(out)
	}
	if !result.IsValid() {
		if out != nil {
			result.PrintErrors(out)
		}
		return false, nil
	}
	return true, nil
}
