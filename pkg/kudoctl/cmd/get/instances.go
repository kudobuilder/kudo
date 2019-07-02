package get

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/xlab/treeprint"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"github.com/kudobuilder/kudo/pkg/kudoctl/util/check"
)

var (
	kubeConfig string
	namespace  string
)

// NewGetInstancesCmd creates a command that lists the instances in the cluster
func NewGetInstancesCmd() *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "instances",
		Short: "Gets all available instances.",
		Long: `
	# Get all available instances
	kudoctl get instances`,
		Run: run,
	}

	getCmd.Flags().StringVar(&kubeConfig, "kubeconfig", "", "The file path to kubernetes configuration file; defaults to $HOME/.kube/config")
	getCmd.Flags().StringVar(&namespace, "namespace", "default", "The namespace where the operator watches for changes.")

	return getCmd
}

func run(cmd *cobra.Command, args []string) {

	_, err := cmd.Flags().GetString("kubeconfig")
	if err != nil {
		log.Printf("Flag Error: %v", err)
	}

	// If the $KUBECONFIG environment variable is set, use that
	if len(os.Getenv("KUBECONFIG")) > 0 {
		kubeConfig = os.Getenv("KUBECONFIG")
	}

	check.KubeConfigLocationOrDefault(kubeConfig)

	if err := check.ValidateKubeConfigPath(kubeConfig); err != nil {
		log.Printf("Could not check kubeconfig path: %v", err)
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
