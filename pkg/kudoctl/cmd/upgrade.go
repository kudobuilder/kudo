package cmd

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/install"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
	"github.com/pkg/errors"
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
		kubectl kudo upgrade flink --instance dev-flink --version 1.1.1

		# By default parameters are all reused from the previous installation, if you need to modify, use -p
		kubectl kudo upgrade flink --instance dev-flink -p param=xxx`
)

type options struct {
	InstanceName   string
	Namespace      string
	PackageVersion string
	Parameters     map[string]string
}

// defaultOptions initializes the install command options to its defaults
var defaultOptions = &options{
	Namespace: "default",
}

// newUpgradeCmd creates the install command for the CLI
func newUpgradeCmd() *cobra.Command {
	options := defaultOptions
	var parameters []string
	upgradeCmd := &cobra.Command{
		Use:     "upgrade <name>",
		Short:   "Upgrade KUDO package.",
		Long:    `Upgrade KUDO package from current version to new version.`,
		Example: upgradeExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Prior to command execution we parse and validate passed parameters
			var err error
			options.Parameters, err = install.GetParameterMap(parameters)
			if err != nil {
				return errors.WithMessage(err, "could not parse parameters")
			}
			return runUpgrade(args, options)
		},
		SilenceUsage: true,
	}

	upgradeCmd.Flags().StringVar(&options.InstanceName, "instance", "", "The instance name.")
	upgradeCmd.Flags().StringArrayVarP(&parameters, "parameter", "p", nil, "The parameter name and value separated by '='")
	upgradeCmd.Flags().StringVar(&options.Namespace, "namespace", defaultOptions.Namespace, "The namespace where the instance you want to upgrade is installed in.")
	upgradeCmd.Flags().StringVarP(&options.PackageVersion, "version", "v", "", "A specific package version on the official repository. When installing from other sources than official repository, version from inside operator.yaml will be used. (default to the most recent)")

	const usageFmt = "Usage:\n  %s\n\nFlags:\n%s"
	upgradeCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(upgradeCmd.OutOrStderr(), usageFmt, upgradeCmd.UseLine(), upgradeCmd.Flags().FlagUsages())
		return nil
	})
	return upgradeCmd
}

func validateCmd(args []string, options *options) error {
	if len(args) != 1 {
		return fmt.Errorf("expecting exactly one argument - name of the package or path to upgrade")
	}
	if options.InstanceName == "" {
		return fmt.Errorf("please use --instance and specify instance name. It cannot be empty")
	}

	return nil
}

func runUpgrade(args []string, options *options) error {
	err := validateCmd(args, options)
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

	return upgrade(crds.OperatorVersion, kc, options)
}

func upgrade(newOv *v1alpha1.OperatorVersion, kc *kudo.Client, options *options) error {
	operatorName := newOv.Spec.Operator.Name
	nextOperatorVersion := newOv.Spec.Version

	// Make sure the instance you want to upgrade exists
	instance, err := kc.GetInstance(options.InstanceName, options.Namespace)
	if err != nil {
		return errors.Wrapf(err, "verifying the instance does not already exist")
	}
	if instance == nil {
		return fmt.Errorf("instance %s in namespace %s does not exist in the cluster", options.InstanceName, options.Namespace)
	}

	// Check OperatorVersion and if upgraded version is higher than current version
	ov, err := kc.GetOperatorVersion(instance.Spec.OperatorVersion.Name, options.Namespace)
	if err != nil {
		return errors.Wrap(err, "retrieving existing operator version")
	}
	if ov == nil {
		return fmt.Errorf("no operator version for this operator installed yet for %s in namespace %s. Please use install command if you want to install new operator into cluster", operatorName, options.Namespace)
	}
	oldVersion, err := semver.NewVersion(ov.Spec.Version)
	if err != nil {
		return errors.Wrapf(err, "when parsing %s as semver", ov.Spec.Version)
	}
	newVersion, err := semver.NewVersion(nextOperatorVersion)
	if err != nil {
		return errors.Wrapf(err, "when parsing %s as semver", nextOperatorVersion)
	}
	if !oldVersion.LessThan(newVersion) {
		return fmt.Errorf("upgraded version %s is the same or smaller as current version %s -> not upgrading", nextOperatorVersion, ov.Spec.Version)
	}

	// install OV
	versionsInstalled, err := kc.OperatorVersionsInstalled(operatorName, options.Namespace)
	if err != nil {
		return errors.Wrap(err, "retrieving existing operator versions")
	}
	if !install.VersionExists(versionsInstalled, nextOperatorVersion) {
		if _, err := kc.InstallOperatorVersionObjToCluster(newOv, options.Namespace); err != nil {
			return errors.Wrapf(err, "failed installing OperatorVersion %s for operator: %s", nextOperatorVersion, operatorName)
		}
	}

	// Change instance to point to the new OV and optionally update parameters
	err = kc.UpdateInstance(options.InstanceName, options.Namespace, newOv.Name, options.Parameters)
	if err != nil {
		return errors.Wrapf(err, "updating instance to point to new operatorversion %s", newOv.Name)
	}
	return nil
}
