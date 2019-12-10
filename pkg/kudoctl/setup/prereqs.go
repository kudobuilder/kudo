package setup

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

// Defines a single prerequisite that is defined as a k8s resource
type k8sResource interface {
	Install(client *kube.Client) error
	Validate(client *kube.Client) error
	AsRuntimeObj() []runtime.Object
}

//Defines the Prerequisites that need to be in place to run the KUDO manager. This includes setting up the kudo-system namespace and service account
type KudoPrerequisite struct {
	Options Options
	prereqs []k8sResource
}

func Prereqs(options Options) KudoPrerequisite {
	return KudoPrerequisite{
		Options: options,
		prereqs: []k8sResource{
			newNamespaceSetup(options),
			newServiceAccountSetup(options),
			newRoleBindingSetup(options),
			newWebHookSetup(options),
		},
	}
}

func (p KudoPrerequisite) Install(client *kube.Client) error {
	for _, prereq := range p.prereqs {
		err := prereq.Install(client)
		if err != nil {
			return fmt.Errorf("failed to install something: %v", err)
		}
	}
	return nil
}

func (p KudoPrerequisite) AsYamlManifests() ([]string, error) {
	var prereqs []runtime.Object

	for _, prereq := range p.prereqs {
		prereqs = append(prereqs, prereq.AsRuntimeObj()...)
	}

	manifests := make([]string, len(prereqs))
	for i, obj := range prereqs {
		o, err := yaml.Marshal(obj)
		if err != nil {
			return []string{}, err
		}
		manifests[i] = string(o)
	}

	return manifests, nil
}
