package get

import (
	"fmt"
	"log"
	"os"

	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/pkg/errors"

	"github.com/xlab/treeprint"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kudobuilder/kudo/pkg/kudoctl/util/check"
)

// Options defines configuration options for the get command
type Options struct {
	KubeConfigPath string
	Namespace      string
}

// DefaultOptions initializes the get command options to its defaults
var DefaultOptions = &Options{
	Namespace: "default",
}

// Run returns the errors associated with cmd env
func Run(args []string, options *Options) error {

	err := validate(args, options)
	if err != nil {
		return err
	}

	kc, err := kudo.NewClient(options.Namespace, options.KubeConfigPath)
	if err != nil {
		return errors.Wrap(err, "creating kudo client")
	}

	p, err := getInstances(kc, options)
	if err != nil {
		log.Printf("Error: %v", err)
	}
	tree := treeprint.New()

	for _, plan := range p {
		tree.AddBranch(plan)
	}
	fmt.Printf("List of current installed instances in namespace \"%s\":\n", options.Namespace)
	fmt.Println(tree.String())
	return err
}

func validate(args []string, options *Options) error {
	if len(args) != 1 {
		return fmt.Errorf("expecting exactly one argument - \"instances\"")
	}

	if args[0] != "instances" {
		return fmt.Errorf("expecting \"instances\" and not \"%s\"", args[0])
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

func getInstances(kc *kudo.Client, options *Options) ([]string, error) {

	instanceList, err := kc.ListInstances(options.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "getting instances")
	}

	return instanceList, nil
}
