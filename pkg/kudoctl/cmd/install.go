package cmd

import (
	"fmt"
	"strings"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/install"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	installExample = `
		The install argument must be a name of the package in the repository, a path to package in *.tar.gz format,
		or a path to an unpacked package directory.

		# Install the most recent Flink package to your cluster.
		kubectl kudo install flink

		# Install framework from a local filesystem
		kubectl kudo install pkg/kudoctl/util/repo/testdata/zk

		# Install framework from tarball on a local filesystem
		kubectl kudo install pkg/kudoctl/util/repo/testdata/zk.tar.gz

		# Install the Kafka package with all of its dependencies to your cluster.
		kubectl kudo install kafka --all-dependencies

		# Specify a package version of Kafka to install to your cluster.
		kubectl kudo install kafka --package-version=0`
)

// getParameterMap takes a slice of parameter strings, parses parameters into a map of keys and values
func getParameterMap(raw []string) (map[string]string, error) {
	var errs []string
	parameters := make(map[string]string)

	for _, a := range raw {
		key, value, err := parseParameter(a)
		if err != nil {
			errs = append(errs, *err)
			continue
		}
		parameters[key] = value
	}

	if errs != nil {
		return nil, errors.New(strings.Join(errs, ", "))
	}

	return parameters, nil
}

// parseParameter does all the parsing logic for an instance of a parameter provided to the command line
// it expects `=` as a delimiter as in key=value.  It separates keys from values as a return.   Any unexpected param will result in a
// detailed error message.
func parseParameter(raw string) (key string, param string, err *string) {

	var errMsg string
	s := strings.SplitN(raw, "=", 2)
	if len(s) < 2 {
		errMsg = fmt.Sprintf("parameter not set: %+v", raw)
	} else if s[0] == "" {
		errMsg = fmt.Sprintf("parameter name can not be empty: %+v", raw)
	} else if s[1] == "" {
		errMsg = fmt.Sprintf("parameter value can not be empty: %+v", raw)
	}
	if errMsg != "" {
		return "", "", &errMsg
	}
	return s[0], s[1], nil
}

// newInstallCmd creates the install command for the CLI
func newInstallCmd() *cobra.Command {
	options := install.DefaultOptions
	var parameters []string
	installCmd := &cobra.Command{
		Use:     "install <name>",
		Short:   "-> Install an official KUDO package.",
		Long:    `Install a KUDO package from local filesystem or the official repo.`,
		Example: installExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Prior to command execution we parse and validate passed parameters
			var err error
			options.Parameters, err = getParameterMap(parameters)
			if err != nil {
				return errors.WithMessage(err, "could not parse parameters")
			}

			return install.Run(cmd, args, options)
		},
		SilenceUsage: true,
	}

	installCmd.Flags().BoolVar(&options.AllDependencies, "all-dependencies", false, "Installs all dependency packages. (default \"false\")")
	installCmd.Flags().BoolVar(&options.AutoApprove, "auto-approve", false, "Skip interactive approval when existing version found. (default \"false\")")
	installCmd.Flags().StringVar(&options.KubeConfigPath, "kubeconfig", "", "The file path to Kubernetes configuration file. (default \"$HOME/.kube/config\")")
	installCmd.Flags().StringVar(&options.InstanceName, "instance", "", "The instance name. (default to Operator name)")
	installCmd.Flags().StringVar(&options.Namespace, "namespace", "default", "The namespace used for the package installation. (default \"default\"")
	installCmd.Flags().StringArrayVarP(&parameters, "parameter", "p", nil, "The parameter name and value separated by '='")
	installCmd.Flags().StringVar(&options.PackageVersion, "package-version", "", "A specific package version on the official GitHub repo. (default to the most recent)")
	installCmd.Flags().BoolVar(&options.SkipInstance, "skip-instance", false, "If set, install will install the Operator and OperatorVersion, but not an instance. (default \"false\")")

	const usageFmt = "Usage:\n  %s\n\nFlags:\n%s"
	installCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(installCmd.OutOrStderr(), usageFmt, installCmd.UseLine(), installCmd.Flags().FlagUsages())
		return nil
	})
	return installCmd
}
