package plan

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"os/user"

	"path/filepath"

	maestrov1alpha1 "github.com/kubernetes-sigs/kubebuilder-maestro/pkg/apis/maestro/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

var (
	instance   string
	kubeConfig string
	namespace  string
)

const (
	defaultConfigPath = ".kube/config"
)

func NewPlanListCmd() *cobra.Command {
	listCmd := &cobra.Command{
		//Args: cobra.ExactArgs(1),
		Use:   "list",
		Short: "Lists all available plans of a particular instance.",
		Long: `
	# View plan status
	maestroctl plan list --instance=<instanceName>`,
		Run: planListCmd,
	}

	listCmd.Flags().StringVar(&instance, "instance", "", "The instance name.")
	listCmd.Flags().StringVar(&kubeConfig, "kubeconfig", "", "The file path to kubernetes configuration file; defaults to $HOME/.kube/config")
	listCmd.Flags().StringVar(&namespace, "namespace", "default", "The namespace where the operator watches for changes.")

	return listCmd
}

func planListCmd(cmd *cobra.Command, args []string) {

	instanceFlag, err := cmd.Flags().GetString("instance")
	if err != nil || instanceFlag == "" {
		log.Printf("Flag Error: %v", err)
	}

	mustKubeConfig()

	kubeConfigFlag, err := cmd.Flags().GetString("kubeconfig")
	if err != nil || instanceFlag == "" {
		log.Printf("Flag Error: %v", err)
	}

	planList(instanceFlag, kubeConfigFlag)
}

func planList(i, k string) ([]string, error) {

	planList := make([]string, 0)

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return nil, err
	}

	//  Create a Dynamic Client to interface with CRDs.
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	instancesGVR := schema.GroupVersionResource{
		Group:    "maestro.k8s.io",
		Version:  "v1alpha1",
		Resource: "instances",
	}

	//  List all of the Virtual Services.
	instObj, err := dynamicClient.Resource(instancesGVR).Namespace(namespace).Get(instance, metav1.GetOptions{})
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}
	mInstObj, err := instObj.MarshalJSON()

	instance := maestrov1alpha1.Instance{}

	err = json.Unmarshal(mInstObj, &instance)
	if err != nil {
		return nil, err
	}

	frameworkVersionNameOfInstance := instance.Spec.FrameworkVersion.Name

	frameworkGVR := schema.GroupVersionResource{
		Group:    "maestro.k8s.io",
		Version:  "v1alpha1",
		Resource: "frameworkversions",
	}

	//  List all of the Virtual Services.
	frameworkObj, err := dynamicClient.Resource(frameworkGVR).Namespace(namespace).Get(frameworkVersionNameOfInstance, metav1.GetOptions{})
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}
	mFrameworkObj, err := frameworkObj.MarshalJSON()

	framework := maestrov1alpha1.FrameworkVersion{}

	err = json.Unmarshal(mFrameworkObj, &framework)
	if err != nil {
		return nil, err
	}

	plans := framework.Spec.Plans

	for planName, _ := range plans {
		planList = append(planList, planName)
	}

	fmt.Printf("%v\n", planList)

	return planList, nil
}

// mustKubeConfig checks if the kubeconfig file exists.
func mustKubeConfig() {
	// if kubeConfig is not specified, search for the default kubeconfig file under the $HOME/.kube/config.
	if len(kubeConfig) == 0 {
		usr, err := user.Current()
		if err != nil {
			fmt.Printf("Error: failed to determine user's home dir: %v", err)
		}
		kubeConfig = filepath.Join(usr.HomeDir, defaultConfigPath)
	}

	_, err := os.Stat(kubeConfig)
	if err != nil && os.IsNotExist(err) {
		fmt.Printf("Error: failed to find the kubeconfig file (%v): %v", kubeConfig, err)
	}
}
