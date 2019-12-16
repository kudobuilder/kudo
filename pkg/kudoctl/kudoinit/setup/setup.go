package setup

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/crd"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/manager"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/prereq"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

// Install uses Kubernetes client to install KUDO.
func Install(client *kube.Client, opts kudoinit.Options, crdOnly bool) error {
	var initSteps []kudoinit.InitStep
	if crdOnly {
		initSteps = []kudoinit.InitStep{
			crd.NewInitializer(),
		}
	} else {
		initSteps = []kudoinit.InitStep{
			crd.NewInitializer(),
			manager.NewInitializer(opts),
			prereq.NewInitializer(opts),
		}
	}

	// Check if all steps are installable
	for _, initStep := range initSteps {
		if err := initStep.PreInstallCheck(client); err != nil {
			return err
		}
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

func AsYamlManifests(opts kudoinit.Options, crdOnly bool) ([]string, error) {
	var manifests []runtime.Object

	crds := crd.NewInitializer().AsArray()
	manifests = append(manifests, crds...)

	if crdOnly {
		return toYaml(manifests)
	}

	prereqs := prereq.NewInitializer(opts).AsArray()
	manifests = append(manifests, prereqs...)

	mgr := manager.NewInitializer(opts).AsArray()
	manifests = append(manifests, mgr...)

	return toYaml(manifests)
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
