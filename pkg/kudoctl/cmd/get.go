package cmd

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/get"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
)

const getExample = `  # Get all available instances
  kubectl kudo get instances 
`

// newGetCmd creates a command that lists the instances in the cluster
func newGetCmd(out io.Writer) *cobra.Command {
	opts := get.CmdOpts{
		Out: out,
	}

	getCmd := &cobra.Command{
		Use:     "get instances",
		Short:   "Gets all available instances.",
		Example: getExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := env.GetClient(&Settings)
			if err != nil {
				return err
			}
			opts.Client = client
			opts.Namespace = Settings.Namespace

			return get.Run(args, opts)
		},
	}

	getCmd.Flags().StringVarP(opts.Output.AsStringPtr(), "output", "o", "", "Output format for command results.")

	return getCmd
}
