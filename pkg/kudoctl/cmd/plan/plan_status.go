package plan

import (
	"encoding/json"
	"fmt"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/check"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/xlab/treeprint"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

type statusOptions struct {
	instance       string
	kubeConfigPath string
	namespace      string
}

var defaultStatusOptions = &statusOptions{}

//NewPlanStatusCmd creates a new command that shows the status of an instance by looking at its current plan
func NewPlanStatusCmd() *cobra.Command {
	options := defaultStatusOptions
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Shows the status of all plans to an particular instance.",
		Long: `
	# View plan status
	kudoctl plan status --instance=<instanceName> --kubeconfig=<$HOME/.kube/config>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd, args, options)
		},
	}

	statusCmd.Flags().StringVar(&options.instance, "instance", "", "The instance name available from 'kubectl get instances'")
	statusCmd.Flags().StringVar(&options.kubeConfigPath, "kubeconfig", "", "The file path to kubernetes configuration file; defaults to $HOME/.kube/config")
	statusCmd.Flags().StringVar(&options.namespace, "namespace", "default", "The namespace where the instance is running.")

	return statusCmd
}

func runStatus(cmd *cobra.Command, args []string, options *statusOptions) error {

	instanceFlag, err := cmd.Flags().GetString("instance")
	if err != nil || instanceFlag == "" {
		return fmt.Errorf("flag Error: Please set instance flag, e.g. \"--instance=<instanceName>\"")
	}

	configPath, err := check.KubeConfigLocationOrDefault(options.kubeConfigPath)
	if err != nil {
		return fmt.Errorf("error when getting default kubeconfig path: %+v", err)
	}
	options.kubeConfigPath = configPath
	if err := check.ValidateKubeConfigPath(options.kubeConfigPath); err != nil {
		return errors.WithMessage(err, "could not check kubeconfig path")
	}

	_, err = cmd.Flags().GetString("kubeconfig")
	if err != nil || instanceFlag == "" {
		return fmt.Errorf("flag Error: Please set kubeconfig flag, e.g. \"--kubeconfig=<$HOME/.kube/config>\"")
	}

	err = planStatus(options)
	if err != nil {
		return fmt.Errorf("client Error: %v", err)
	}
	return nil
}

func planStatus(options *statusOptions) error {

	tree := treeprint.New()

	config, err := clientcmd.BuildConfigFromFlags("", options.kubeConfigPath)
	if err != nil {
		return err
	}

	//  Create a Dynamic Client to interface with CRDs.
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	instancesGVR := schema.GroupVersionResource{
		Group:    "kudo.k8s.io",
		Version:  "v1alpha1",
		Resource: "instances",
	}

	instObj, err := dynamicClient.Resource(instancesGVR).Namespace(options.namespace).Get(options.instance, metav1.GetOptions{})
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
		Group:    "kudo.k8s.io",
		Version:  "v1alpha1",
		Resource: "operatorversions",
	}

	//  List all of the Virtual Services.
	operatorObj, err := dynamicClient.Resource(operatorGVR).Namespace(options.namespace).Get(operatorVersionNameOfInstance, metav1.GetOptions{})
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

	planExecutionsGVR := schema.GroupVersionResource{
		Group:    "kudo.k8s.io",
		Version:  "v1alpha1",
		Resource: "planexecutions",
	}

	activePlanObj, err := dynamicClient.Resource(planExecutionsGVR).Namespace(options.namespace).Get(instance.Status.ActivePlan.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	mPlanObj, err := activePlanObj.MarshalJSON()
	if err != nil {
		return err
	}

	activePlanType := kudov1alpha1.PlanExecution{}

	err = json.Unmarshal(mPlanObj, &activePlanType)
	if err != nil {
		return err
	}

	rootDisplay := fmt.Sprintf("%s (Operator-Version: \"%s\" Active-Plan: \"%s\")", instance.Name, instance.Spec.OperatorVersion.Name, instance.Status.ActivePlan.Name)
	rootBranchName := tree.AddBranch(rootDisplay)

	for name, plan := range operator.Spec.Plans {
		if name == activePlanType.Spec.PlanName {
			planDisplay := fmt.Sprintf("Plan %s (%s strategy) [%s]", name, plan.Strategy, activePlanType.Status.State)
			planBranchName := rootBranchName.AddBranch(planDisplay)
			for _, phase := range activePlanType.Status.Phases {
				phaseDisplay := fmt.Sprintf("Phase %s (%s strategy) [%s]", phase.Name, phase.Strategy, phase.State)
				phaseBranchName := planBranchName.AddBranch(phaseDisplay)
				for _, steps := range phase.Steps {
					stepsDisplay := fmt.Sprintf("Step %s (%s)", steps.Name, steps.State)
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

	fmt.Printf("Plan(s) for \"%s\" in namespace \"%s\":\n", instance.Name, options.namespace)
	fmt.Println(tree.String())

	return nil
}
