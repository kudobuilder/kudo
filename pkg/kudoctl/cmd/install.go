package cmd

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/install"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var (
	installExample = `  The install argument must be a name of the package in the repository, a path to package in *.tgz format,
  or a path to an unpacked package directory.

  # Install the most recent Flink package to your cluster.
  kubectl kudo install flink
  #*Note*: should you have a local  "flink" folder in the current directory it will take precedence over the remote repository.

  # Install operator from a local filesystem
  kubectl kudo install pkg/kudoctl/util/repo/testdata/zk

  # Install operator from tarball on a local filesystem
  kubectl kudo install pkg/kudoctl/util/repo/testdata/zk.tgz

  # Install operator from tarball at URL
  kubectl kudo install http://kudo.dev/zk.tgz

  # Specify a package version of Kafka to install to your cluster
  kubectl kudo install kafka --version=1.1.1`
)

// newInstallCmd creates the install command for the CLI
func newInstallCmd(fs afero.Fs) *cobra.Command {
	options := install.DefaultOptions
	var parameters []string
	installCmd := &cobra.Command{
		Use:     "install <name>",
		Short:   "Install an official KUDO package.",
		Long:    `Install a KUDO package from local filesystem or the official repo.`,
		Example: installExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Prior to command execution we parse and validate passed arguments
			var err error
			options.Parameters, err = install.GetParameterMap(parameters)
			if err != nil {
				return errors.WithMessage(err, "could not parse arguments")
			}

			return install.Run(args, options, fs, &Settings)
		},
	}

	installCmd.Flags().StringVar(&options.InstanceName, "instance", "", "The instance name. (default to Operator name)")
	installCmd.Flags().StringArrayVarP(&parameters, "parameter", "p", nil, "The parameter name and value separated by '='")
	installCmd.Flags().StringVarP(&options.PackageVersion, "version", "v", "", "A specific package version on the official GitHub repo. (default to the most recent)")
	installCmd.Flags().StringVar(&options.RepoName, "repo", "", "Name of repository configuration to use. (default defined by context)")
	installCmd.Flags().BoolVar(&options.SkipInstance, "skip-instance", false, "If set, install will install the Operator and OperatorVersion, but not an instance. (default \"false\")")
	return installCmd
}
