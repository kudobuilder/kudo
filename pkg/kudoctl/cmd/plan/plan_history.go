package plan

import (
	"encoding/json"
	"fmt"
	"time"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/env"
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
		fmt.Printf("History of all plan-executions for instance \"%s\" in namespace \"%s\":\n", options.Instance, options.Namespace)
		labelSelector = "instance=" + options.Instance
	} else {
		fmt.Printf("History of plan-executions for instance \"%s\" in namespace \"%s\" to operator-version \"%s\":\n", options.Instance, options.Namespace, args[0])
		labelSelector = "operator-version=" + args[0] + ", instance=" + options.Instance
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

	planExecutionList := kudov1alpha1.PlanExecutionList{}

	err = json.Unmarshal(mInstObj, &planExecutionList)
	if err != nil {
		return err
	}

	tree := treeprint.New()

	if len(planExecutionList.Items) == 0 {
		fmt.Printf("No history found for \"%s\" in namespace \"%s\".\n", options.Instance, options.Namespace)
	} else {
		for _, i := range planExecutionList.Items {
			duration := time.Since(i.CreationTimestamp.Time)
			historyDisplay := fmt.Sprintf("%s (created %v ago)", i.Name, duration.Round(time.Second))
			tree.AddBranch(historyDisplay)
		}

		fmt.Println(tree.String())
	}

	return nil
}
