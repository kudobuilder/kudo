package cmd

import (
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/diagnostics"
)

const (
	diagCollectExample = `  # collect diagnostics example
  kubectl kudo diagnostics collect --instance=%instance% --namespace=%namespace%
`
)

func newDiagnosticsCmd(fs afero.Fs) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diagnostics",
		Short: "diagnostics",
		Long:  "diagnostics command has sub-commands to collect and analyze diagnostics data",
	}
	cmd.AddCommand(newDiagnosticsCollectCmd(fs))
	return cmd
}

func newDiagnosticsCollectCmd(fs afero.Fs) *cobra.Command {
	var logSince time.Duration
	var instance string
	cmd := &cobra.Command{
		Use:     "collect",
		Short:   "collect",
		Long:    "collect diagnostics",
		Example: diagCollectExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			return diagnostics.Collect(fs, toDiagOpts(instance, logSince), &Settings)
		},
	}
	cmd.Flags().StringVar(&instance, "instance", "", "The instance name.")
	cmd.Flags().DurationVar(&logSince, "log.since", 0, "Only return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs. Only one of since-time / since may be used.")

	return cmd
}

func toDiagOpts(instance string, logSince time.Duration) *diagnostics.Options {
	opts := diagnostics.Options{Instance: instance}
	if logSince > 0 {
		sec := int64(logSince.Round(time.Second).Seconds())
		opts.LogSince = sec
	}
	return &opts
}
