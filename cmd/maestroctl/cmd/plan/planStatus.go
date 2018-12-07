package plan

import (
	"encoding/json"
	"fmt"
	maestrov1alpha1 "github.com/kubernetes-sigs/kubebuilder-maestro/pkg/apis/maestro/v1alpha1"
	"github.com/spf13/cobra"
	"github.com/xlab/treeprint"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

func NewPlanStatusCmd() *cobra.Command {
	statusCmd := &cobra.Command{
		Args:  cobra.ExactArgs(1),
		Use:   "status",
		Short: "Shows the status of a particular plan of an instance.",
		Long: `
	# View plan status
	maestroctl plan status <planName> --instance=<instanceName>`,
		Run: planStatusCmd,
	}

	statusCmd.Flags().StringVar(&instance, "instance", "", "The instance name available from 'kubectl get instances'")
	statusCmd.Flags().StringVar(&kubeConfig, "kubeconfig", "", "The file path to kubernetes configuration file; defaults to $HOME/.kube/config")
	statusCmd.Flags().StringVar(&namespace, "namespace", "default", "The namespace where the instance is running.")

	return statusCmd
}

func planStatusCmd(cmd *cobra.Command, args []string) {

	instanceFlag, err := cmd.Flags().GetString("instance")
	if err != nil || instanceFlag == "" {
		log.Printf("Flag Error: %v", err)
	}

	mustKubeConfig()

	kubeConfigFlag, err := cmd.Flags().GetString("kubeconfig")
	if err != nil || instanceFlag == "" {
		log.Printf("Flag Error: %v", err)
	}

	planStatus(instanceFlag, kubeConfigFlag, args[0])
}

func planStatus(i, k, a string) error {

	tree := treeprint.New()

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return err
	}

	//  Create a Dynamic Client to interface with CRDs.
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	instancesGVR := schema.GroupVersionResource{
		Group:    "maestro.k8s.io",
		Version:  "v1alpha1",
		Resource: "instances",
	}

	instObj, err := dynamicClient.Resource(instancesGVR).Namespace(namespace).Get(instance, metav1.GetOptions{})
	if err != nil {
		log.Printf("Error: %v", err)
		return err
	}
	mInstObj, err := instObj.MarshalJSON()

	instance := maestrov1alpha1.Instance{}

	err = json.Unmarshal(mInstObj, &instance)
	if err != nil {
		return err
	}

	activePlanNameOfInstance := instance.Status.ActivePlan.Name

	frameworkGVR := schema.GroupVersionResource{
		Group:    "maestro.k8s.io",
		Version:  "v1alpha1",
		Resource: "planexecutions",
	}

	activePlanObj, err := dynamicClient.Resource(frameworkGVR).Namespace(namespace).Get(activePlanNameOfInstance, metav1.GetOptions{})
	if err != nil {
		log.Printf("Error: %v", err)
		return err
	}
	mPlanObj, err := activePlanObj.MarshalJSON()

	activePlanType := maestrov1alpha1.PlanExecution{}

	err = json.Unmarshal(mPlanObj, &activePlanType)
	if err != nil {
		return err
	}

	activePlan := activePlanType.Status

	planDisplay := fmt.Sprintf("%s (%s strategy) [%s]", activePlan.Name, activePlan.Strategy, activePlan.State)

	planBranchName := tree.AddBranch(planDisplay)

	for _, phase := range activePlan.Phases {
		phaseDisplay := fmt.Sprintf("%s (%s strategy) [%s]", phase.Name, phase.Strategy, phase.State)

		phaseBranchName := planBranchName.AddBranch(phaseDisplay)

		for _, step := range phase.Steps {
			stepDisplay := fmt.Sprintf("%s [%s]", step.Name, step.State)

			phaseBranchName.AddNode(stepDisplay)
		}
	}

	fmt.Println(tree.String())

	return nil
}
