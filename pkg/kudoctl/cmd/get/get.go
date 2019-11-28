package get

import (
	"errors"
	"fmt"
	"log"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"

	"github.com/xlab/treeprint"
)

// Run returns the errors associated with cmd env
func Run(args []string, settings *env.Settings) error {

	err := validate(args)
	if err != nil {
		return err
	}

	kc, err := env.GetClient(settings)
	if err != nil {
		return fmt.Errorf("creating kudo client: %w", err)
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
		return errors.New(`expecting exactly one argument - "instances"`)
	}

	if args[0] != "instances" {
		return fmt.Errorf(`expecting "instances" and not %q`, args[0])
	}

	return nil

}

func getInstances(kc *kudo.Client, settings *env.Settings) ([]string, error) {

	instanceList, err := kc.ListInstances(settings.Namespace)
	if err != nil {
		return nil, fmt.Errorf("getting instances: %w", err)
	}

	return instanceList, nil
}
