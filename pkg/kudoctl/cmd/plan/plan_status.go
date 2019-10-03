package plan

import (
	"encoding/json"
	"fmt"
	"log"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/spf13/cobra"
	"github.com/xlab/treeprint"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

// DefaultStatusOptions provides the default options for plan status
var DefaultStatusOptions = &Options{}

// RunStatus runs the plan status command
func RunStatus(cmd *cobra.Command, args []string, options *Options, settings *env.Settings) error {

	instanceFlag, err := cmd.Flags().GetString("instance")
	if err != nil || instanceFlag == "" {
		return fmt.Errorf("flag Error: Please set instance flag, e.g. \"--instance=<instanceName>\"")
	}

	err = planStatus(options, settings)
	if err != nil {
		return fmt.Errorf("client Error: %v", err)
	}
	return nil
}

func planStatus(options *Options, settings *env.Settings) error {

	tree := treeprint.New()

	config, err := clientcmd.BuildConfigFromFlags("", settings.KubeConfig)
	if err != nil {
		return err
	}

	//  Create a Dynamic Client to interface with CRDs.
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	instancesGVR := schema.GroupVersionResource{
		Group:    "kudo.dev",
		Version:  "v1alpha1",
		Resource: "instances",
	}

	instObj, err := dynamicClient.Resource(instancesGVR).Namespace(options.Namespace).Get(options.Instance, metav1.GetOptions{})
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

	operatorVersionNameOfInstance := instance.Spec.OperatorVersion.Name

	operatorGVR := schema.GroupVersionResource{
		Group:    "kudo.dev",
		Version:  "v1alpha1",
		Resource: "operatorversions",
	}

	//  List all of the Virtual Services.
	operatorObj, err := dynamicClient.Resource(operatorGVR).Namespace(options.Namespace).Get(operatorVersionNameOfInstance, metav1.GetOptions{})
	if err != nil {
		return err
	}

	mOperatorObj, err := operatorObj.MarshalJSON()
	if err != nil {
		return err
	}

	operator := kudov1alpha1.OperatorVersion{}

	err = json.Unmarshal(mOperatorObj, &operator)
	if err != nil {
		return err
	}

	activePlanStatus := instance.GetPlanInProgress()

	if activePlanStatus == nil {
		log.Printf("No active plan exists for instance %s", instance.Name)
		return nil
	}

	rootDisplay := fmt.Sprintf("%s (Operator-Version: \"%s\" Active-Plan: \"%s\")", instance.Name, instance.Spec.OperatorVersion.Name, activePlanStatus.Name)
	rootBranchName := tree.AddBranch(rootDisplay)

	for name, plan := range operator.Spec.Plans {
		if name == activePlanStatus.Name {
			planDisplay := fmt.Sprintf("Plan %s (%s strategy) [%s]", name, plan.Strategy, activePlanStatus.Status)
			planBranchName := rootBranchName.AddBranch(planDisplay)
			for _, phase := range activePlanStatus.Phases {
				phaseDisplay := fmt.Sprintf("Phase %s [%s]", phase.Name, phase.Status)
				phaseBranchName := planBranchName.AddBranch(phaseDisplay)
				for _, steps := range phase.Steps {
					stepsDisplay := fmt.Sprintf("Step %s (%s)", steps.Name, steps.Status)
					phaseBranchName.AddBranch(stepsDisplay)
				}
			}
		} else {
			planDisplay := fmt.Sprintf("Plan %s (%s strategy) [NOT ACTIVE]", name, plan.Strategy)
			planBranchName := rootBranchName.AddBranch(planDisplay)
			for _, phase := range plan.Phases {
				phaseDisplay := fmt.Sprintf("Phase %s (%s strategy) [NOT ACTIVE]", phase.Name, phase.Strategy)
				phaseBranchName := planBranchName.AddBranch(phaseDisplay)
				for _, steps := range plan.Phases {
					stepsDisplay := fmt.Sprintf("Step %s (%s strategy) [NOT ACTIVE]", steps.Name, steps.Strategy)
					stepBranchName := phaseBranchName.AddBranch(stepsDisplay)
					for _, step := range steps.Steps {
						stepDisplay := fmt.Sprintf("%s [NOT ACTIVE]", step.Name)
						stepBranchName.AddBranch(stepDisplay)
					}
				}
			}
		}
	}

	fmt.Printf("Plan(s) for \"%s\" in namespace \"%s\":\n", instance.Name, options.Namespace)
	fmt.Println(tree.String())

	return nil
}
