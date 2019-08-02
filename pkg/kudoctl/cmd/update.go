package cmd

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/install"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	updateExample = `
		The update argument must be a name of the instance.

		# Update dev-flink instance with setting parameter param with value value
		kubectl kudo update dev-flink -p param=value

		# Update dev-flink instance in namespace services with setting parameter param with value value
		kubectl kudo update dev-flink -n services -p param=value`
)

type updateOptions struct {
	Namespace  string
	Parameters map[string]string
}

// defaultOptions initializes the install command options to its defaults
var defaultUpdateOptions = &updateOptions{
	Namespace: "default",
}

// newUpdateCmd creates the install command for the CLI
func newUpdateCmd() *cobra.Command {
	options := defaultUpdateOptions
	var parameters []string
	updateCmd := &cobra.Command{
		Use:     "update <instance-name>",
		Short:   "Update installed KUDO operator.",
		Long:    `Update installed KUDO operator with new parameters.`,
		Example: updateExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Prior to command execution we parse and validate passed parameters
			var err error
			options.Parameters, err = install.GetParameterMap(parameters)
			if err != nil {
				return errors.WithMessage(err, "could not parse parameters")
			}
			return runUpdate(args, options)
		},
	}

	updateCmd.Flags().StringArrayVarP(&parameters, "parameter", "p", nil, "The parameter name and value separated by '='")
	updateCmd.Flags().StringVar(&options.Namespace, "namespace", defaultOptions.Namespace, "The namespace where the instance you want to upgrade is installed in.")
	return updateCmd
}

func validateUpdateCmd(args []string, options *updateOptions) error {
	if len(args) != 1 {
		return errors.New("expecting exactly one argument - name of the instance installed in your cluster")
	}
	if len(options.Parameters) == 0 {
		return errors.New("Need to specify at least one parameter to override via -p otherwise there is nothing to update")
	}

	return nil
}

func runUpdate(args []string, options *updateOptions) error {
	err := validateUpdateCmd(args, options)
	if err != nil {
		return err
	}
	instanceToUpdate := args[0]

	kc, err := kudo.NewClient(options.Namespace, viper.GetString("kubeconfig"))
	if err != nil {
		return errors.Wrap(err, "creating kudo client")
	}

	return update(instanceToUpdate, kc, options)
}

func update(instanceToUpdate string, kc *kudo.Client, options *updateOptions) error {
	// Make sure the instance you want to upgrade exists
	instance, err := kc.GetInstance(instanceToUpdate, options.Namespace)
	if err != nil {
		return errors.Wrapf(err, "verifying the instance does not already exist")
	}
	if instance == nil {
		return fmt.Errorf("instance %s in namespace %s does not exist in the cluster", instanceToUpdate, options.Namespace)
	}

	// Update parameters
	err = kc.UpdateInstance(instanceToUpdate, options.Namespace, nil, options.Parameters)
	if err != nil {
		return errors.Wrapf(err, "updating instance %s", instanceToUpdate)
	}
	fmt.Printf("Instance %s was updated ヽ(•‿•)ノ", instanceToUpdate)
	return nil
}
