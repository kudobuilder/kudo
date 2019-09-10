package get

import (
	"fmt"
	"log"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"

	"github.com/pkg/errors"
	"github.com/xlab/treeprint"
)

// Run returns the errors associated with cmd env
func Run(args []string, settings *env.Settings) error {

	err := validate(args)
	if err != nil {
		return err
	}

	kc, err := kudo.NewClient(settings.Namespace, settings.KubeConfig)
	if err != nil {
		return errors.Wrap(err, "creating kudo client")
	}

	p, err := getInstances(kc, settings)
	if err != nil {
		log.Printf("Error: %v", err)
	}
	tree := treeprint.New()

	for _, plan := range p {
		tree.AddBranch(plan)
	}
	fmt.Printf("List of current installed instances in namespace \"%s\":\n", settings.Namespace)
	fmt.Println(tree.String())
	return err
}

func validate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expecting exactly one argument - \"instances\"")
	}

	if args[0] != "instances" {
		return fmt.Errorf("expecting \"instances\" and not \"%s\"", args[0])
	}

	return nil

}

func getInstances(kc *kudo.Client, settings *env.Settings) ([]string, error) {

	instanceList, err := kc.ListInstances(settings.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "getting instances")
	}

	return instanceList, nil
}
