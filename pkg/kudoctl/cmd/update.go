package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/params"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

var (
	updateDesc = `Update KUDO operator instance with new arguments. The update does not accept any arguments.
`
	updateExample = `  # Update dev-flink instance with setting parameter param with value value
  kubectl kudo update --instance dev-flink -p param=value

  # Update dev-flink instance in namespace services with setting parameter param with value value
  kubectl kudo update --instance dev-flink -n services -p param=value`
)

type updateOptions struct {
	InstanceName string
	Parameters   map[string]string
	Wait         bool
	WaitTime     int64
}

// defaultOptions initializes the install command options to its defaults
var defaultUpdateOptions = &updateOptions{}

// newUpdateCmd creates the install command for the CLI
func newUpdateCmd() *cobra.Command {
	options := defaultUpdateOptions
	var parameters []string
	var parameterFiles []string
	updateCmd := &cobra.Command{
		Use:     "update",
		Short:   "Update KUDO operator instance.",
		Long:    updateDesc,
		Example: updateExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Prior to command execution we parse and validate passed arguments
			var err error
			options.Parameters, err = params.GetParameterMap(fs, parameters, parameterFiles)
			if err != nil {
				return fmt.Errorf("could not parse parameters: %v", err)
			}
			return runUpdate(args, options, &Settings)
		},
	}

	updateCmd.Flags().StringVar(&options.InstanceName, "instance", "", "The instance name.")
	updateCmd.Flags().StringArrayVarP(&parameters, "parameter", "p", nil, "The parameter name and value separated by '='")
	updateCmd.Flags().StringArrayVarP(&parameterFiles, "parameter-file", "P", nil, "YAML file with parameters")
	updateCmd.Flags().BoolVar(&options.Wait, "wait", false, "Specify if the CLI should wait for the update to complete before returning (default \"false\")")
	updateCmd.Flags().Int64Var(&options.WaitTime, "wait-time", 300, "Specify the max wait time in seconds for CLI for the update to complete before returning (default \"300\")")

	return updateCmd
}

func validateUpdateCmd(args []string, options *updateOptions) error {
	if len(args) != 0 {
		return errors.New("expecting no arguments provided for update. Only named flags are accepted")
	}
	if options.InstanceName == "" {
		return errors.New("--instance flag has to be provided to indicate which instance you want to update")
	}
	if len(options.Parameters) == 0 {
		return errors.New("need to specify at least one parameter to override via -p otherwise there is nothing to update")
	}

	return nil
}

func runUpdate(args []string, options *updateOptions, settings *env.Settings) error {
	err := validateUpdateCmd(args, options)
	if err != nil {
		return err
	}
	instanceToUpdate := options.InstanceName

	kc, err := env.GetClient(settings)
	if err != nil {
		return fmt.Errorf("creating kudo client: %w", err)
	}

	return update(instanceToUpdate, kc, options, settings)
}

func update(instanceToUpdate string, kc *kudo.Client, options *updateOptions, settings *env.Settings) error {
	// Make sure the instance you want to upgrade exists
	instance, err := kc.GetInstance(instanceToUpdate, settings.Namespace)
	if err != nil {
		return fmt.Errorf("verifying the instance does not already exist: %w", err)
	}
	if instance == nil {
		return fmt.Errorf("instance %s in namespace %s does not exist in the cluster", instanceToUpdate, settings.Namespace)
	}

	// Update arguments
	err = kc.UpdateInstance(instanceToUpdate, settings.Namespace, nil, options.Parameters, nil, options.Wait, time.Duration(options.WaitTime)*time.Second)
	if err != nil {
		return fmt.Errorf("updating instance %s %w", instanceToUpdate, err)
	}
	clog.Printf("Instance %s was updated.", instanceToUpdate)
	return nil
}
