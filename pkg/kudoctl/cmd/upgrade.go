package cmd

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/install"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
	"github.com/pkg/errors"
	"github.com/rogpeppe/go-internal/semver"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	upgradeExample = `
		The upgrade argument must be a name of the package in the repository, a path to package in *.tar.gz format,
		or a path to an unpacked package directory.

		# Upgrade flink instance dev-flink to the latest version
		kubectl kudo upgrade flink --instance dev-flink
		
		*Note*: should you have a local "flink" folder in the current directory it will take precedence over the remote repository.

		# Upgrade flink to the version 1.1.1
		kubectl kudo upgrade flink --instance dev-flink --version 1.1.1`
)

type options struct {
	InstanceName   string
	Namespace      string
	PackageVersion string
}

// defaultOptions initializes the install command options to its defaults
var defaultOptions = &options{
	Namespace: "default",
}

// newUpgradeCmd creates the install command for the CLI
func newUpgradeCmd() *cobra.Command {
	options := defaultOptions
	upgradeCmd := &cobra.Command{
		Use:     "upgrade <name>",
		Short:   "-> Upgrade KUDO package.",
		Long:    `Upgrade KUDO package from current version to new version.`,
		Example: installExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgrade(args, options)
		},
		SilenceUsage: true,
	}

	upgradeCmd.Flags().StringVar(&options.InstanceName, "instance", "", "The instance name. (default to Operator name)")
	upgradeCmd.Flags().StringVar(&options.Namespace, "namespace", "default", "The where the instance you want to upgrade is installed in. (default \"default\"")
	upgradeCmd.Flags().StringVarP(&options.PackageVersion, "version", "v", "", "A specific package version on the official GitHub repo. (default to the most recent)")

	const usageFmt = "Usage:\n  %s\n\nFlags:\n%s"
	upgradeCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(upgradeCmd.OutOrStderr(), usageFmt, upgradeCmd.UseLine(), upgradeCmd.Flags().FlagUsages())
		return nil
	})
	return upgradeCmd
}

func validate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expecting exactly one argument - name of the package or path to upgrade")
	}

	return nil
}

func runUpgrade(args []string, options *options) error {
	err := validate(args)
	if err != nil {
		return err
	}
	packageToUpgrade := args[0]

	kc, err := kudo.NewClient(options.Namespace, viper.GetString("kubeconfig"))
	if err != nil {
		return errors.Wrap(err, "creating kudo client")
	}

	// Resolve the package to upgrade to
	repository, err := repo.NewOperatorRepository(repo.Default)
	if err != nil {
		return errors.WithMessage(err, "could not build operator repository")
	}
	crds, err := install.GetPackageCRDs(packageToUpgrade, options.PackageVersion, repository)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve package CRDs for operator: %s", packageToUpgrade)
	}
	operatorName := crds.Operator.ObjectMeta.Name
	nextOperatorVersion := crds.OperatorVersion.Spec.Version

	// Make sure the instance you want to upgrade exists
	instance, err := kc.GetInstance(options.InstanceName, options.Namespace)
	if err != nil {
		return errors.Wrapf(err, "verifying the instance does not already exist")
	}
	if instance == nil {
		return fmt.Errorf("instance %s in namespace %s you want to upgrade does not exist in the cluster", options.InstanceName, options.Namespace)
	}

	// Check OperatorVersion and if upgraded version is higher than current version
	ov, err := kc.GetOperatorVersion(instance.Spec.OperatorVersion.Name, instance.Spec.OperatorVersion.Namespace)
	if err != nil {
		return errors.Wrap(err, "retrieving existing operator version")
	}
	if ov == nil {
		return fmt.Errorf("no operator version for this operator installed yet for %s in namespace %s. Please use install command if you want to install new operator into cluster", operatorName, options.Namespace)
	}
	compareVersions := semver.Compare(ov.Spec.Version, nextOperatorVersion)
	if compareVersions == 0 {
		return fmt.Errorf("upgraded version %s is the same as current version %s -> not upgrading", ov.Spec.Version, nextOperatorVersion)
	}
	if compareVersions > 0 {
		return fmt.Errorf("you're trying to upgrade from version %s to version %s which is not upgrade but downgrade -> not upgrading", ov.Spec.Version, nextOperatorVersion)
	}

	// install OV
	if _, err := kc.InstallOperatorVersionObjToCluster(crds.OperatorVersion, options.Namespace); err != nil {
		return errors.Wrapf(err, "failed installing OperatorVersion for operator: %s", operatorName)
	}

	// Change instance to point to the new OV
	err = kc.UpdateOperatorVersion(options.InstanceName, options.Namespace, crds.OperatorVersion.Name)
	if err != nil {
		return errors.Wrapf(err, "updating instance to point to new operatorversion %s", crds.OperatorVersion.Name)
	}
	return nil
}
