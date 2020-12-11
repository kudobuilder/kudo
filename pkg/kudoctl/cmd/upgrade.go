package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/install"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/params"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	pkgresolver "github.com/kudobuilder/kudo/pkg/kudoctl/packages/resolver"
	deps "github.com/kudobuilder/kudo/pkg/kudoctl/resources/dependencies"
	"github.com/kudobuilder/kudo/pkg/kudoctl/resources/upgrade"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
)

var (
	upgradeDesc = `Upgrade KUDO package from current version to new version. The upgrade argument must be a name of the 
package in the repository, a path to package in *.tgz format, or a path to an unpacked package directory.`
	upgradeExample = `  # Upgrade flink instance dev-flink to the latest version
  kubectl kudo upgrade flink --instance dev-flink
  *Note*: should you have a local "flink" folder in the current directory it will take precedence over the remote repository.

  # Upgrade flink to the version 1.1.1
  kubectl kudo upgrade flink --instance dev-flink --operator-version 1.1.1

  # By default arguments are all reused from the previous installation, if you need to modify, use -p
  kubectl kudo upgrade flink --instance dev-flink -p param=xxx`
)

type options struct {
	install.RepositoryOptions
	InstanceName    string
	AppVersion      string
	OperatorVersion string
	Parameters      map[string]string
}

// defaultOptions initializes the install command options to its defaults
var defaultOptions = &options{}

// newUpgradeCmd creates the install command for the CLI
func newUpgradeCmd(fs afero.Fs) *cobra.Command {
	options := defaultOptions
	var parameters []string
	var parameterFiles []string
	upgradeCmd := &cobra.Command{
		Use:     "upgrade <name>",
		Short:   "Upgrade KUDO package.",
		Long:    upgradeDesc,
		Example: upgradeExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Prior to command execution we parse and validate passed arguments
			var err error
			options.Parameters, err = params.GetParameterMap(fs, parameters, parameterFiles)
			if err != nil {
				return fmt.Errorf("could not parse parameters: %v", err)
			}
			return runUpgrade(args, options, fs, &Settings)
		},
	}

	upgradeCmd.Flags().StringVar(&options.InstanceName, "instance", "", "The instance name.")
	upgradeCmd.Flags().StringArrayVarP(&parameters, "parameter", "p", nil, "The parameter name and value separated by '='")
	upgradeCmd.Flags().StringArrayVarP(&parameterFiles, "parameter-file", "P", nil, "YAML file with parameters")
	upgradeCmd.Flags().StringVar(&options.RepoName, "repo", "", "Name of repository configuration to use. (default defined by context)")
	upgradeCmd.Flags().StringVar(&options.AppVersion, "app-version", "",
		"A specific app version in the official repository. When installing from other sources than an official repository, a version from inside operator.yaml will be used. (default to the most recent)")
	upgradeCmd.Flags().StringVar(&options.OperatorVersion, "operator-version", "",
		"A specific operator version in the official repository. When installing from other sources than an official repository, a version from inside operator.yaml will be used. (default to the most recent)")

	return upgradeCmd
}

func validateCmd(args []string, options *options) error {
	if len(args) != 1 {
		return errors.New("expecting exactly one argument - name of the package or path to upgrade")
	}
	if options.InstanceName == "" {
		return errors.New("please use --instance and specify instance name. It cannot be empty")
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
		return fmt.Errorf("creating kudo client: %w", err)
	}

	// Resolve the package to upgrade to
	repository, err := repo.ClientFromSettings(fs, settings.Home, options.RepoName)
	if err != nil {
		return fmt.Errorf("could not build operator repository: %w", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %v", err)
	}

	resolver := pkgresolver.NewPackageResolver(repository, wd)
	pr, err := resolver.Resolve(packageToUpgrade, options.AppVersion, options.OperatorVersion)
	if err != nil {
		return fmt.Errorf("failed to resolve operator package for: %s: %w", packageToUpgrade, err)
	}

	pr.Resources.OperatorVersion.SetNamespace(settings.Namespace)

	dependencies, err := deps.Resolve(pr.Resources.OperatorVersion, pr.DependenciesResolver)
	if err != nil {
		return err
	}

	return upgrade.OperatorVersion(kc, pr.Resources.OperatorVersion, options.InstanceName, options.Parameters, dependencies)
}
