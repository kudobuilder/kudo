package cmd

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/install"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
	util "github.com/kudobuilder/kudo/pkg/util/kudo"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var (
	upgradeDesc = `Upgrade KUDO package from current version to new version. The upgrade argument must be a name of the 
package in the repository, a path to package in *.tgz format, or a path to an unpacked package directory.`
	upgradeExample = `  # Upgrade flink instance dev-flink to the latest version
  kubectl kudo upgrade flink --instance dev-flink
  *Note*: should you have a local "flink" folder in the current directory it will take precedence over the remote repository.

  # Upgrade flink to the version 1.1.1
  kubectl kudo upgrade flink --instance dev-flink --version 1.1.1

  # By default arguments are all reused from the previous installation, if you need to modify, use -p
  kubectl kudo upgrade flink --instance dev-flink -p param=xxx`
)

type options struct {
	install.RepositoryOptions
	InstanceName   string
	PackageVersion string
	Parameters     map[string]string
}

// defaultOptions initializes the install command options to its defaults
var defaultOptions = &options{}

// newUpgradeCmd creates the install command for the CLI
func newUpgradeCmd(fs afero.Fs) *cobra.Command {
	options := defaultOptions
	var parameters []string
	upgradeCmd := &cobra.Command{
		Use:     "upgrade <name>",
		Short:   "Upgrade KUDO package.",
		Long:    upgradeDesc,
		Example: upgradeExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Prior to command execution we parse and validate passed arguments
			var err error
			options.Parameters, err = install.GetParameterMap(parameters)
			if err != nil {
				return errors.WithMessage(err, "could not parse arguments")
			}
			return runUpgrade(args, options, fs, &Settings)
		},
	}

	upgradeCmd.Flags().StringVar(&options.InstanceName, "instance", "", "The instance name.")
	upgradeCmd.Flags().StringArrayVarP(&parameters, "parameter", "p", nil, "The parameter name and value separated by '='")
	upgradeCmd.Flags().StringVar(&options.RepoName, "repo", "", "Name of repository configuration to use. (default defined by context)")
	upgradeCmd.Flags().StringVar(&options.PackageVersion, "version", "", "A specific package version on the official repository. When installing from other sources than official repository, version from inside operator.yaml will be used. (default to the most recent)")

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

func runUpgrade(args []string, options *options, fs afero.Fs, settings *env.Settings) error {
	err := validateCmd(args, options)
	if err != nil {
		return err
	}
	packageToUpgrade := args[0]

	kc, err := env.GetClient(settings)
	if err != nil {
		return errors.Wrap(err, "creating kudo client")
	}

	// Resolve the package to upgrade to
	repository, err := repo.ClientFromSettings(fs, settings.Home, options.RepoName)
	if err != nil {
		return errors.WithMessage(err, "could not build operator repository")
	}
	crds, err := install.GetPackageCRDs(packageToUpgrade, options.PackageVersion, repository)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve package CRDs for operator: %s", packageToUpgrade)
	}

	return upgrade(crds.OperatorVersion, kc, options, settings)
}

func upgrade(newOv *v1alpha1.OperatorVersion, kc *kudo.Client, options *options, settings *env.Settings) error {
	operatorName := newOv.Spec.Operator.Name
	nextOperatorVersion := newOv.Spec.Version

	// Make sure the instance you want to upgrade exists
	instance, err := kc.GetInstance(options.InstanceName, settings.Namespace)
	if err != nil {
		return errors.Wrapf(err, "verifying the instance does not already exist")
	}
	if instance == nil {
		return fmt.Errorf("instance %s in namespace %s does not exist in the cluster", options.InstanceName, settings.Namespace)
	}

	// Check OperatorVersion and if upgraded version is higher than current version
	ov, err := kc.GetOperatorVersion(instance.Spec.OperatorVersion.Name, settings.Namespace)
	if err != nil {
		return errors.Wrap(err, "retrieving existing operator version")
	}
	if ov == nil {
		return fmt.Errorf("no operator version for this operator installed yet for %s in namespace %s. Please use install command if you want to install new operator into cluster", operatorName, settings.Namespace)
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
	versionsInstalled, err := kc.OperatorVersionsInstalled(operatorName, settings.Namespace)
	if err != nil {
		return errors.Wrap(err, "retrieving existing operator versions")
	}
	if !install.VersionExists(versionsInstalled, nextOperatorVersion) {
		if _, err := kc.InstallOperatorVersionObjToCluster(newOv, settings.Namespace); err != nil {
			return errors.Wrapf(err, "failed installing OperatorVersion %s for operator: %s", nextOperatorVersion, operatorName)
		}
		fmt.Printf("operatorversion.%s/%s successfully created\n", newOv.APIVersion, newOv.Name)
	}

	// Change instance to point to the new OV and optionally update arguments
	err = kc.UpdateInstance(options.InstanceName, settings.Namespace, util.String(newOv.Name), options.Parameters)
	if err != nil {
		return errors.Wrapf(err, "updating instance to point to new operatorversion %s", newOv.Name)
	}
	fmt.Printf("instance.%s/%s successfully updated\n", instance.APIVersion, instance.Name)
	return nil
}
