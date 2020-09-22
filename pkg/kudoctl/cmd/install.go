package cmd

import (
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/install"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/params"
)

var (
	installExample = `  The install argument must be a name of the package in the repository, a URL or path to package in *.tgz format,
  or a path to an unpacked package directory.

  # Install the most recent Flink package from KUDO repository to your cluster.
  kubectl kudo install flink

  # Install operator from a local filesystem
  kubectl kudo install pkg/kudoctl/util/repo/testdata/zk

  # Install operator from tarball on a local filesystem
  kubectl kudo install pkg/kudoctl/util/repo/testdata/zk.tgz

  # Install operator from tarball at URL
  kubectl kudo install http://kudo.dev/zk.tgz

  # Install operator from an in-cluster operator version
  kubectl kudo install zookeeper --operator-version=0.3.0 --in-cluster

  # Specify an operator version of Kafka to install to your cluster
  kubectl kudo install kafka --operator-version=1.1.1`
)

// newInstallCmd creates the install command for the CLI
func newInstallCmd(fs afero.Fs) *cobra.Command {
	options := install.DefaultOptions
	var parameters []string
	var parameterFiles []string
	installCmd := &cobra.Command{
		Use:     "install <name>",
		Short:   "Install an official KUDO package.",
		Long:    `Install a KUDO package from local filesystem or the official repo.`,
		Example: installExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Prior to command execution we parse and validate passed arguments
			var err error
			options.Parameters, err = params.GetParameterMap(fs, parameters, parameterFiles)
			if err != nil {
				return fmt.Errorf("could not parse parameters: %v", err)
			}

			return install.Run(args, options, fs, &Settings)
		},
	}

	installCmd.Flags().StringVar(&options.InstanceName, "instance", "", "The Instance name. (defaults to Operator name appended with -instance)")
	installCmd.Flags().StringArrayVarP(&parameters, "parameter", "p", nil, "The parameter name and value separated by '='")
	installCmd.Flags().StringArrayVarP(&parameterFiles, "parameter-file", "P", nil, "YAML file with parameters")
	installCmd.Flags().StringVar(&options.RepoName, "repo", "", "Name of repository configuration to use. (default defined by context)")
	installCmd.Flags().StringVar(&options.AppVersion, "app-version", "", "A specific app version in the official GitHub repo. (default to the most recent)")
	installCmd.Flags().StringVar(&options.OperatorVersion, "operator-version", "", "A specific operator version int the official GitHub repo. (default to the most recent)")
	installCmd.Flags().BoolVar(&options.SkipInstance, "skip-instance", false, "If set, install will install the Operator and OperatorVersion, but not an Instance. (default \"false\")")
	installCmd.Flags().BoolVar(&options.Wait, "wait", false, "Specify if the CLI should wait for the install to complete before returning (default \"false\")")
	installCmd.Flags().Int64Var(&options.WaitTime, "wait-time", 300, "Specify the max wait time in seconds for CLI for the install to complete before returning (default \"300\")")
	installCmd.Flags().BoolVar(&options.CreateNameSpace, "create-namespace", false, "If set, install will create the specified namespace and will fail if it exists. (default \"false\")")
	installCmd.Flags().BoolVar(&options.InCluster, "in-cluster", false, "Specify if the CLI should resolve the package using the operator version already installed in the cluster. (default \"false\")")

	return installCmd
}
