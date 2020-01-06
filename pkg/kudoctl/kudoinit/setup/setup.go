package setup

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/crd"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/manager"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/prereq"
)

// Install uses Kubernetes client to install KUDO.
func Install(client *kube.Client, opts kudoinit.Options, crdOnly bool) error {
	initSteps := initSteps(opts, crdOnly)

	result := kudoinit.NewResult()
	// Check if all steps are installable
	for _, initStep := range initSteps {
		result.Merge(initStep.PreInstallCheck(client))
	}

	result.PrintWarnings(os.Stdout)
	if !result.IsValid() {
		result.PrintErrors(os.Stdout)
		return nil
	}

	// Install everything
	for _, initStep := range initSteps {
		if err := initStep.Install(client); err != nil {
			return fmt.Errorf("%s: %v", initStep.Description(), err)
		}
		clog.Printf("✅ installed %s", initStep.Description())
	}

	return nil
}

func initSteps(opts kudoinit.Options, crdOnly bool) []kudoinit.InitStep {
	if crdOnly {
		return []kudoinit.InitStep{
			crd.NewInitializer(),
		}
	}

	return []kudoinit.InitStep{
		crd.NewInitializer(),
		prereq.NewInitializer(opts),
		manager.NewInitializer(opts),
	}
}

func AsYamlManifests(opts kudoinit.Options, crdOnly bool) ([]string, error) {
	initSteps := initSteps(opts, crdOnly)
	var allManifests []runtime.Object

	for _, initStep := range initSteps {
		allManifests = append(allManifests, initStep.AsArray()...)
	}

	return toYaml(allManifests)
}

func toYaml(objs []runtime.Object) ([]string, error) {
	manifests := make([]string, len(objs))
	for i, obj := range objs {
		o, err := yaml.Marshal(obj)
		if err != nil {
			return []string{}, err
		}
		manifests[i] = string(o)
	}

	return manifests, nil
}
