package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

const (
	uninstallExample = `kubectl kudo uninstall --instance flink`
)

type uninstallOptions struct {
	InstanceName string
	Wait         bool
	WaitTime     int64
}

type uninstallCmd struct{}

func (cmd *uninstallCmd) run(options uninstallOptions, settings *env.Settings) error {
	kc, err := env.GetClient(settings)
	clog.V(3).Printf("acquiring kudo client")
	if err != nil {
		clog.V(3).Printf("failed to acquire kudo client: %v", err)
		return fmt.Errorf("failed to acquire kudo client: %w", err)
	}

	return cmd.uninstall(kc, options, settings)
}

func (cmd *uninstallCmd) uninstall(kc *kudo.Client, options uninstallOptions, settings *env.Settings) error {
	instance, err := kc.GetInstance(options.InstanceName, settings.Namespace)
	if err != nil {
		return fmt.Errorf("failed to verify if instance already exists: %w", err)
	}

	if instance == nil {
		return fmt.Errorf("instance %s in namespace %s does not exist in the cluster", options.InstanceName, settings.Namespace)
	}

	err = kc.DeleteInstance(options.InstanceName, settings.Namespace)
	if err != nil {
		return err
	}

	if options.Wait {
		waitDuration := time.Duration(options.WaitTime) * time.Second
		if err := kc.WaitForInstanceDeleted(options.InstanceName, settings.Namespace, waitDuration); err != nil {
			return err
		}
	}

	clog.Printf("instance.%s/%s deleted\n", instance.APIVersion, options.InstanceName)
	return nil
}

func newUninstallCmd() *cobra.Command {
	options := uninstallOptions{}
	uninstall := &uninstallCmd{}

	uninstallCmd := &cobra.Command{
		Use:     "uninstall",
		Short:   "Uninstall a KUDO package.",
		Long:    "Uninstall the instance of a KUDO package. This also removes dependent objects, e.g. deployments, pods",
		Example: uninstallExample,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("the command expects no arguments and --instance flag must be provided.\n %s", cmd.UsageString())
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return uninstall.run(options, &Settings)
		},
	}

	uninstallCmd.Flags().StringVar(&options.InstanceName, "instance", "", "The instance name.")
	if err := uninstallCmd.MarkFlagRequired("instance"); err != nil {
		panic(err)
	}

	uninstallCmd.Flags().BoolVar(&options.Wait, "wait", false, "Specify if the CLI should wait for the uninstall to complete before returning (default \"false\")")
	uninstallCmd.Flags().Int64Var(&options.WaitTime, "wait-time", 300, "Specify the max wait time in seconds for CLI for the uninstall to complete before returning (default \"300\")")

	return uninstallCmd
}
