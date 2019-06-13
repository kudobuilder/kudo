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
		# Install the most recent Flink package to your cluster.
		kubectl kudo install flink

		# Install the Kafka package with all of its dependencies to your cluster.
		kubectl kudo install kafka --all-dependencies

		# Specify a package version of Kafka to install to your cluster.
		kubectl kudo install kafka --package-version=0`
)

// parseParameters parse raw parameter strings into a map of keys and values
func parseParameters(raw []string, parameters map[string]string) error {
	var errs []string

	for _, a := range raw {
		// Using '=' as the delimiter. Split after the first delimiter to support using '=' in values
		s := strings.SplitN(a, "=", 2)
		if len(s) < 2 {
			errs = append(errs, fmt.Sprintf("parameter not set: %+v", a))
			continue
		}
		if s[0] == "" {
			errs = append(errs, fmt.Sprintf("parameter name can not be empty: %+v", a))
			continue
		}
		if s[1] == "" {
			errs = append(errs, fmt.Sprintf("parameter value can not be empty: %+v", a))
			continue
		}
		parameters[s[0]] = s[1]
	}

	if errs != nil {
		return errors.New(strings.Join(errs, ", "))
	}

	return nil
}

// newInstallCmd creates the install command for the CLI
func newInstallCmd() *cobra.Command {
	options := install.DefaultOptions
	var parameters []string
	installCmd := &cobra.Command{
		Use:     "install <name>",
		Short:   "-> Install an official KUDO package.",
		Long:    `Install a KUDO package from the official GitHub repo.`,
		Example: installExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Prior to command execution we parse and validate passed parameters
			options.Parameters = make(map[string]string)
			if err := parseParameters(parameters, options.Parameters); err != nil {
				return errors.WithMessage(err, "could not parse parameters")
			}

			return install.Run(cmd, args, options)
		},
		SilenceUsage: true,
	}

	installCmd.Flags().BoolVar(&options.AllDependencies, "all-dependencies", false, "Installs all dependency packages. (default \"false\")")
	installCmd.Flags().BoolVar(&options.AutoApprove, "auto-approve", false, "Skip interactive approval when existing version found. (default \"false\")")
	installCmd.Flags().StringVar(&options.KubeConfigPath, "kubeconfig", "", "The file path to Kubernetes configuration file. (default \"$HOME/.kube/config\")")
	installCmd.Flags().StringVar(&options.InstanceName, "instance", "", "The instance name. (default to Framework name)")
	installCmd.Flags().StringVar(&options.Namespace, "namespace", "default", "The namespace used for the package installation. (default \"default\"")
	installCmd.Flags().StringArrayVarP(&parameters, "parameter", "p", nil, "The parameter name and value separated by '='")
	installCmd.Flags().StringVar(&options.PackageVersion, "package-version", "", "A specific package version on the official GitHub repo. (default to the most recent)")

	const usageFmt = "Usage:\n  %s\n\nFlags:\n%s"
	installCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(installCmd.OutOrStderr(), usageFmt, installCmd.UseLine(), installCmd.Flags().FlagUsages())
		return nil
	})
	return installCmd
}
