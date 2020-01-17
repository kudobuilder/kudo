package plan

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/xlab/treeprint"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
)

// Options are the configurable options for plans
type Options struct {
	Out      io.Writer
	Instance string
}

var (
	// DefaultHistoryOptions provides the default options for plan history
	DefaultHistoryOptions = &Options{}
)

// RunHistory runs the plan history command
func RunHistory(cmd *cobra.Command, options *Options, settings *env.Settings) error {
	instanceFlag, err := cmd.Flags().GetString("instance")
	if err != nil || instanceFlag == "" {
		return errors.New(`flag Error: Please set instance flag, e.g. "--instance=<instanceName>"`)
	}

	err = planHistory(options, settings)
	if err != nil {
		return fmt.Errorf("client Error: %v", err)
	}
	return nil
}

func planHistory(options *Options, settings *env.Settings) error {
	namespace := settings.Namespace

	kc, err := env.GetClient(settings)
	if err != nil {
		clog.Printf("Unable to create KUDO client to talk to kubernetes API server: %v", err)
		return err
	}
	instance, err := kc.GetInstance(options.Instance, namespace)
	if err != nil {
		return err
	}
	if instance == nil {
		return clog.Errorf("instance %s/%s does not exist", namespace, options.Instance)
	}

	tree := treeprint.New()
	timeLayout := "2006-01-02T15:04:05"

	for _, p := range instance.Status.PlanStatus {
		msg := "never run" // this is for the cases when status was not yet populated

		if !p.LastFinishedRun.IsZero() { // plan already finished
			t := p.LastFinishedRun.Format(timeLayout)
			msg = fmt.Sprintf("last finished run at %s (%s)", t, string(p.Status))
		} else if p.Status.IsRunning() {
			msg = "is running"
		} else if p.Status != "" {
			msg = string(p.Status)
		}
		historyDisplay := fmt.Sprintf("%s (%s)", p.Name, msg)
		tree.AddBranch(historyDisplay)
	}

	fmt.Println(tree.String())

	return nil
}
