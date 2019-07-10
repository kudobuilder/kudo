package get

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"log"
	"os"

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

// Options defines configuration options for the install command
type Options struct {
	InstanceName   string
	KubeConfigPath string
	Namespace      string
	Parameters     map[string]string
	PackageVersion string
	SkipInstance   bool
}

// DefaultOptions initializes the install command options to its defaults
var DefaultOptions = &Options{
	Namespace: "default",
}

func Run(args []string, options *Options) error {

	err := validate(args, options)
	if err != nil {
		return err
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
	return err
}

func validate(args []string, options *Options) error {
	if len(args) != 1 {
		return fmt.Errorf("expecting exactly one argument - name of the package or path to install")
	}

	// If the $KUBECONFIG environment variable is set, use that
	if len(os.Getenv("KUBECONFIG")) > 0 {
		options.KubeConfigPath = os.Getenv("KUBECONFIG")
	}

	configPath, err := check.KubeConfigLocationOrDefault(options.KubeConfigPath)
	if err != nil {
		return fmt.Errorf("error when getting default kubeconfig path: %+v", err)
	}
	options.KubeConfigPath = configPath
	if err := check.ValidateKubeConfigPath(options.KubeConfigPath); err != nil {
		return errors.WithMessage(err, "could not check kubeconfig path")
	}
	_, err = clientcmd.BuildConfigFromFlags("", options.KubeConfigPath)
	if err != nil {
		return errors.Wrap(err, "getting config failed")
	}

	return nil

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
