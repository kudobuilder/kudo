package plan

import (
	"encoding/json"
	"fmt"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/spf13/cobra"
	"github.com/xlab/treeprint"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
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
func RunHistory(cmd *cobra.Command, args []string, options *Options, settings *env.Settings) error {

	instanceFlag, err := cmd.Flags().GetString("instance")
	if err != nil || instanceFlag == "" {
		return fmt.Errorf("flag Error: Please set instance flag, e.g. \"--instance=<instanceName>\"")
	}

	err = planHistory(args, options, settings)
	if err != nil {
		return fmt.Errorf("client Error: %v", err)
	}
	return nil
}

func planHistory(args []string, options *Options, settings *env.Settings) error {

	config, err := clientcmd.BuildConfigFromFlags("", settings.KubeConfig)
	if err != nil {
		return err
	}

	// Create a Dynamic Client to interface with CRDs.
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	planExecutionsGVR := schema.GroupVersionResource{
		Group:    "kudo.dev",
		Version:  "v1alpha1",
		Resource: "planexecutions",
	}

	var labelSelector string
	if len(args) == 0 {
		fmt.Printf("History of all plan executions for instance \"%s\" in namespace \"%s\":\n", options.Instance, options.Namespace)
		labelSelector = fmt.Sprintf("instance=%s", options.Instance)
	} else {
		fmt.Printf("History of plan-executions for instance \"%s\" in namespace \"%s\" for operator-version \"%s\":\n", options.Instance, options.Namespace, args[0])
		labelSelector = fmt.Sprintf("operator-version=%s, instance=%s", args[0], options.Instance)
	}

	instObj, err := dynamicClient.Resource(planExecutionsGVR).Namespace(options.Namespace).List(metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return err
	}

	mInstObj, err := instObj.MarshalJSON()
	if err != nil {
		return err
	}

	instance := kudov1alpha1.Instance{}

	err = json.Unmarshal(mInstObj, &instance)
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
