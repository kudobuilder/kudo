package cmd

import (
	"errors"
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/install"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	pkgresolver "github.com/kudobuilder/kudo/pkg/kudoctl/packages/resolver"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"

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
				return fmt.Errorf("could not parse arguments: %w", err)
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
	resolver := pkgresolver.New(repository)
	pkg, err := resolver.Resolve(packageToUpgrade, options.PackageVersion)
	if err != nil {
		return fmt.Errorf("failed to resolve package CRDs for operator: %s: %w", packageToUpgrade, err)
	}

	resources := pkg.Resources

	return kudo.UpgradeOperatorVersion(kc, resources.OperatorVersion, options.InstanceName, settings.Namespace, options.Parameters)
}
