package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/plan"
)

const (
	planHistoryExample = `  # View plan history
  kubectl kudo plan history --instance=<instanceName>
`
	planStatusExample = `  # View plan status
  kubectl kudo plan status --instance=<instanceName>
`
	planTriggerExample = `  # Trigger an instance plan
kubectl kudo plan trigger <planName> --instance=<instanceName>
`
)

// newPlanCmd creates a new command that shows the plans available for an instance
func newPlanCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "View all available plans.",
		Long:  `The plan command has subcommands to view all available plans.`,
	}

	cmd.AddCommand(NewPlanHistoryCmd())
	cmd.AddCommand(NewPlanStatusCmd(out))
	cmd.AddCommand(NewPlanTriggerCmd())

	return cmd
}

// NewPlanHistoryCmd creates a command that shows the plan history of an instance.
func NewPlanHistoryCmd() *cobra.Command {
	options := plan.DefaultHistoryOptions
	cmd := &cobra.Command{
		Use:     "history",
		Short:   "Lists history for each plan of an instance.",
		Example: planHistoryExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return plan.RunHistory(cmd, options, &Settings)
		},
	}

	cmd.Flags().StringVar(&options.Instance, "instance", "", "The instance name available from 'kubectl get instances'")

	return cmd
}

// NewPlanStatusCmd creates a new command that shows the status of an instance by looking at its current plan
func NewPlanStatusCmd(out io.Writer) *cobra.Command {
	options := &plan.StatusOptions{Out: out}
	cmd := &cobra.Command{
		Use:     "status",
		Short:   "Shows the status of all plans to an particular instance.",
		Example: planStatusExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return plan.Status(options, &Settings)
		},
	}

	cmd.Flags().StringVar(&options.Instance, "instance", "", "The instance name available from 'kubectl get instances'")
	cmd.Flags().BoolVar(&options.Wait, "wait", false, "Specify if the CLI should wait for the plan to complete before returning (default \"false\")")
	cmd.Flags().StringVarP(options.Output.AsStringPtr(), "output", "o", "", "Output format for command results.")

	if err := cmd.MarkFlagRequired("instance"); err != nil {
		clog.Printf("Please choose the instance with '--instance=<instanceName>': %v", err)
		os.Exit(1)
	}

	return cmd
}

// NewPlanTriggerCmd creates a command that triggers a specific plan for an instance.
func NewPlanTriggerCmd() *cobra.Command {
	options := &plan.TriggerOptions{}
	cmd := &cobra.Command{
		Use:     "trigger",
		Short:   "Triggers a specific plan on a particular instance.",
		Example: planTriggerExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return plan.RunTrigger(options, &Settings)
		},
	}

	cmd.Flags().StringVar(&options.Instance, "instance", "", "The instance name available from 'kubectl get instances'")
	cmd.Flags().StringVar(&options.Plan, "name", "", "The plan name")
	cmd.Flags().BoolVar(&options.Wait, "wait", false, "Specify if the CLI should wait for the plan to complete before returning (default \"false\")")
	cmd.Flags().Int64Var(&options.WaitTime, "wait-time", 300, "Specify the max wait time in seconds for CLI for the plan to complete before returning (default \"300\")")

	return cmd
}
