package plan

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/spf13/cobra"
	"github.com/xlab/treeprint"
)

// Options are the configurable options for plans
type Options struct {
	Instance  string
	Namespace string
}

var (
	// DefaultHistoryOptions provides the default options for plan history
	DefaultHistoryOptions = &Options{}
)

// RunHistory runs the plan history command
func RunHistory(cmd *cobra.Command, options *Options, settings *env.Settings) error {
	instanceFlag, err := cmd.Flags().GetString("instance")
	if err != nil || instanceFlag == "" {
		return fmt.Errorf("flag Error: Please set instance flag, e.g. \"--instance=<instanceName>\"")
	}

	err = planHistory(options, settings)
	if err != nil {
		return fmt.Errorf("client Error: %v", err)
	}
	return nil
}

func planHistory(options *Options, settings *env.Settings) error {
	kc, err := kudo.NewClient(settings.Namespace, settings.KubeConfig)
	if err != nil {
		fmt.Printf("Unable to create kudo client to talk to kubernetes API server %w", err)
		return err
	}
	instance, err := kc.GetInstance(options.Instance, options.Namespace)
	if errors.IsNotFound(err) {
		fmt.Printf("Instance %s/%s does not exist", instance.Namespace, instance.Name)
	}
	if err != nil {
		return err
	}

	tree := treeprint.New()
	timeLayout := "2006-01-02"

	for _, p := range instance.Status.PlanStatus {
		msg := "never run"
		if p.Status != "" && !p.LastFinishedRun.IsZero() { // plan already finished
			t := p.LastFinishedRun.Format(timeLayout)
			msg = fmt.Sprintf("last run at %s", t)
		} else if p.Status.IsRunning() {
			msg = "is running"
		}
		historyDisplay := fmt.Sprintf("%s (%s)", p.Name, msg)
		tree.AddBranch(historyDisplay)
	}

	fmt.Println(tree.String())

	return nil
}
