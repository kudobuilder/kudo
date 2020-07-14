package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/diagnostics"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

const (
	DiagDir            = "diag"
	diagCollectExample = `  # collect diagnostics example
  kubectl kudo diagnostics collect --instance flink
`
)

func newDiagnosticsCmd(fs afero.Fs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diagnostics",
		Short: "collect diagnostics",
		Long:  "diagnostics provides functionality to collect and analyze diagnostics data",
	}
	cmd.AddCommand(newDiagnosticsCollectCmd(fs))
	return cmd
}

func newDiagnosticsCollectCmd(fs afero.Fs) *cobra.Command {
	var logSince time.Duration
	var instance string
	var outputDirectory string
	cmd := &cobra.Command{
		Use:     "collect",
		Short:   "collect diagnostics",
		Long:    "collect data relevant for diagnostics of the provided instance's state",
		Example: diagCollectExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := kudo.NewClient(Settings.KubeConfig, Settings.RequestTimeout, Settings.Validate)
			if err != nil {
				return fmt.Errorf("failed to create kudo client: %v", err)
			}
			return diagnostics.Collect(fs, instance, outputDirectory, diagnostics.NewOptions(logSince), c, &Settings)
		},
	}
	cmd.Flags().StringVar(&instance, "instance", "", "The instance name.")
	cmd.Flags().StringVar(&outputDirectory, "output-directory", DiagDir, "Directory in which diagnostics output will be saved. It is expected to not exist.")
	cmd.Flags().DurationVar(&logSince, "log-since", 0, "Only return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs.")

	return cmd
}
