package get

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"

	"github.com/spf13/cobra"
	"github.com/xlab/treeprint"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"

	"path/filepath"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
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

// NewGetInstancesCmd creates a command that lists the instances in the cluster
func NewGetInstancesCmd() *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "instances",
		Short: "Gets all available instances.",
		Long: `
	# Get all available instances
	kudoctl get instances`,
		Run: instancesGetCmd,
	}

	getCmd.Flags().StringVar(&kubeConfig, "kubeconfig", "", "The file path to kubernetes configuration file; defaults to $HOME/.kube/config")
	getCmd.Flags().StringVar(&namespace, "namespace", "default", "The namespace where the operator watches for changes.")

	return getCmd
}

func instancesGetCmd(cmd *cobra.Command, args []string) {

	mustKubeConfig()

	_, err := cmd.Flags().GetString("kubeconfig")
	if err != nil {
		log.Printf("Flag Error: %v", err)
	}

	p, err := getInstances()
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

func getInstances() ([]string, error) {

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
		Group:    "kudo.k8s.io",
		Version:  "v1alpha1",
		Resource: "instances",
	}

	instObj, err := dynamicClient.Resource(instancesGVR).Namespace(namespace).List(metav1.ListOptions{})
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, err
	}

	mInstObj, _ := instObj.MarshalJSON()

	instance := kudov1alpha1.InstanceList{}

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
