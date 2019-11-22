package cmd

import (
	"io"
	"os"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/plan"
	"github.com/spf13/cobra"
)

const (
	planHistExample = `  # View plan status
  kubectl kudo plan history <operatorVersion> --instance=<instanceName>
`
	planStatuExample = `  # View plan status
  kubectl kudo plan status --instance=<instanceName>
`
)

// newPlanCmd creates a new command that shows the plans available for an instance
func newPlanCmd(out io.Writer) *cobra.Command {
	newCmd := &cobra.Command{
		Use:   "plan",
		Short: "View all available plans.",
		Long:  `The plan command has subcommands to view all available plans.`,
	}

	newCmd.AddCommand(NewPlanHistoryCmd())
	newCmd.AddCommand(NewPlanStatusCmd(out))

	return newCmd
}

// NewPlanHistoryCmd creates a command that shows the plan history of an instance.
func NewPlanHistoryCmd() *cobra.Command {
	options := plan.DefaultHistoryOptions
	listCmd := &cobra.Command{
		Use:     "history",
		Short:   "Lists history to a specific operator-version of an instance.",
		Example: planHistExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return plan.RunHistory(cmd, options, &Settings)
		},
	}

	listCmd.Flags().StringVar(&options.Instance, "instance", "", "The instance name.")

	return listCmd
}

//NewPlanStatusCmd creates a new command that shows the status of an instance by looking at its current plan
func NewPlanStatusCmd(out io.Writer) *cobra.Command {
	options := &plan.Options{Out: out}
	statusCmd := &cobra.Command{
		Use:     "status",
		Short:   "Shows the status of all plans to an particular instance.",
		Example: planStatuExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return plan.Status(options, &Settings)
		},
	}

	statusCmd.Flags().StringVar(&options.Instance, "instance", "", "The instance name available from 'kubectl get instances'")
	if err := statusCmd.MarkFlagRequired("instance"); err != nil {
		clog.Printf("failed to make --instance flag as required: %v", err)
		os.Exit(1)
	}

	return statusCmd
}
