package cmd

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"

	"github.com/spf13/cobra"
)

const (
	uninstallExample = `kubectl kudo uninstall --instance flink`
)

type uninstallOptions struct {
	InstanceName string
}

type uninstallCmd struct{}

func (cmd *uninstallCmd) run(options uninstallOptions, settings *env.Settings) error {
	kc, err := kudo.NewClient(settings.Namespace, settings.KubeConfig)
	clog.V(3).Printf("acquiring kudo client")
	if err != nil {
		clog.V(3).Printf("failed to acquire kudo client: %v", err)
		return fmt.Errorf("failed to acquire kudo client: %w", err)
	}

	return cmd.uninstall(kc, options.InstanceName, settings)
}

func (cmd *uninstallCmd) uninstall(kc *kudo.Client, instanceName string, settings *env.Settings) error {
	instance, err := kc.GetInstance(instanceName, settings.Namespace)
	if err != nil {
		return fmt.Errorf("failed to verify if instance already exists: %w", err)
	}

	if instance == nil {
		return fmt.Errorf("instance %s in namespace %s does not exist in the cluster", instanceName, settings.Namespace)
	}

	err = kc.DeleteInstance(instanceName, settings.Namespace)
	if err != nil {
		return err
	}

	clog.Printf("instance.%s/%s deleted\n", instance.APIVersion, instanceName)
	return nil
}

func newUninstallCmd() *cobra.Command {
	options := uninstallOptions{}
	uninstall := &uninstallCmd{}

	uninstallCmd := &cobra.Command{
		Use:     "uninstall",
		Short:   "Uninstall a KUDO package.",
		Long:    "Uninstall the instance of a KUDO package.",
		Example: uninstallExample,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return uninstall.run(options, &Settings)
		},
	}

	uninstallCmd.Flags().StringVar(&options.InstanceName, "instance", "", "The instance name.")
	if err := uninstallCmd.MarkFlagRequired("instance"); err != nil {
		panic(err)
	}

	return uninstallCmd
}
