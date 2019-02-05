package list

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/xlab/treeprint"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"os/user"

	"path/filepath"

	maestrov1alpha1 "github.com/universal-operator/universal-operator/pkg/apis/maestro/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

var (
	kubeConfig string
	namespace  string
)

const (
	defaultConfigPath = ".kube/config"
)

func NewListInstancesCmd() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "instances",
		Short: "Lists all available instances.",
		Long: `
	# List all available instances
	maestroctl list instances`,
		Run: instancesListCmd,
	}

	listCmd.Flags().StringVar(&kubeConfig, "kubeconfig", "", "The file path to kubernetes configuration file; defaults to $HOME/.kube/config")
	listCmd.Flags().StringVar(&namespace, "namespace", "default", "The namespace where the operator watches for changes.")

	return listCmd
}

func instancesListCmd(cmd *cobra.Command, args []string) {

	mustKubeConfig()

	_, err := cmd.Flags().GetString("kubeconfig")
	if err != nil {
		log.Printf("Flag Error: %v", err)
	}

	p, err := listInstances()
	if err != nil {
		log.Printf("Error: %v", err)
	}
	tree := treeprint.New()

	for _, plan := range p {
		tree.AddBranch(plan)
	}
	fmt.Printf("List of current instances in namespace \"%s\":\n", namespace)
	fmt.Println(tree.String())

}

func listInstances() ([]string, error) {

	instanceList := make([]string, 0)

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

	instObj, err := dynamicClient.Resource(instancesGVR).Namespace(namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	mInstObj, err := instObj.MarshalJSON()

	instance := maestrov1alpha1.InstanceList{}

	//log.Println(instObj)

	err = json.Unmarshal(mInstObj, &instance)
	if err != nil {
		return nil, err
	}

	for _, i := range instance.Items {
		instanceList = append(instanceList, i.Name)
	}

	return instanceList, nil
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
