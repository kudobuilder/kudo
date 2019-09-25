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
	Purge        bool
}

type uninstallCmd struct{}

func (cmd *uninstallCmd) run(options uninstallOptions, settings *env.Settings) error {
	kc, err := kudo.NewClient(settings.Namespace, settings.KubeConfig)
	clog.V(3).Printf("acquiring kudo client")
	if err != nil {
		clog.V(3).Printf("failed to acquire kudo client")
		return fmt.Errorf("failed to acquire kudo client: %w", err)
	}

	return cmd.uninstall(kc, options.InstanceName, options.Purge, settings)
}

func (cmd *uninstallCmd) uninstall(kc *kudo.Client, instanceName string, purge bool, settings *env.Settings) error {
	instance, err := kc.GetInstance(instanceName, settings.Namespace)
	if err != nil {
		return fmt.Errorf("failed to verify if instance already exists: %w", err)
	}

	if instance == nil {
		return fmt.Errorf("instance %s in namespace %s does not exist in the cluster", instanceName, settings.Namespace)
	}

	operatorVersionName := instance.Spec.OperatorVersion.Name
	operatorVersionNamespace := settings.Namespace
	if instance.Spec.OperatorVersion.Namespace != "" {
		operatorVersionNamespace = instance.Spec.OperatorVersion.Namespace
	}

	err = kc.DeleteInstance(instanceName, settings.Namespace)
	if err != nil {
		return err
	}

	if purge {
		err = cmd.deleteOperatorVersionAndOperator(kc, operatorVersionName, operatorVersionNamespace)
		if err != nil {
			return fmt.Errorf("failed to delete operatorversion and operator of instance %s: %w", instanceName, err)
		}
	}

	return nil
}

func (cmd *uninstallCmd) deleteOperatorVersionAndOperator(kc *kudo.Client, operatorVersionName, namespace string) error {
	operatorVersion, err := kc.GetOperatorVersion(operatorVersionName, namespace)
	if err != nil {
		return fmt.Errorf("failed to get operatorversion %s in namespace %s: %w", operatorVersionName, namespace, err)
	}

	if operatorVersion == nil {
		return fmt.Errorf("operatorversion %s in namespace %s does not exist in the cluster", operatorVersionName, namespace)
	}

	operatorName := operatorVersion.Spec.Operator.Name
	operatorNamespace := namespace
	if operatorVersion.Spec.Operator.Namespace != "" {
		operatorNamespace = operatorVersion.Spec.Operator.Namespace
	}

	err = kc.DeleteOperatorVersion(operatorVersionName, namespace)
	if err != nil {
		return err
	}

	err = cmd.deleteOperator(kc, operatorName, operatorNamespace)
	if err != nil {
		return fmt.Errorf("failed to delete operator of operatorversion %s: %w", operatorVersionName, err)
	}

	return nil
}

func (cmd *uninstallCmd) deleteOperator(kc *kudo.Client, operatorName, namespace string) error {
	operator, err := kc.GetOperator(operatorName, namespace)
	if err != nil {
		return fmt.Errorf("failed to get operator %s in namespace %s: %w", operatorName, namespace, err)
	}

	if operator == nil {
		return fmt.Errorf("operator %s in namespace %s does not exist in the cluster", operatorName, namespace)
	}

	return kc.DeleteOperator(operatorName, namespace)
}

func newUninstallCmd() *cobra.Command {
	options := uninstallOptions{}
	uninstall := &uninstallCmd{}

	uninstallCmd := &cobra.Command{
		Use:     "uninstall",
		Short:   "Uninstall a KUDO package.",
		Long:    "Uninstall the instance and/or operator and operatorversion of a KUDO package.",
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
	uninstallCmd.Flags().BoolVar(&options.Purge, "purge", false, "If set, the Operator and OperatorVersion of the instance will be removed as well. (default \"false\")")

	return uninstallCmd
}
