package setup

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/crd"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/manager"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/prereq"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
)

// Install uses Kubernetes client to install KUDO.
func Install(client *kube.Client, opts kudoinit.Options, crdOnly bool) error {

	clog.Printf("✅ installing crds")
	if err := crd.NewInitializer().Install(client); err != nil {
		return fmt.Errorf("crds: %v", err)
	}
	if crdOnly {
		return nil
	}

	clog.Printf("✅ preparing service accounts and other requirements for controller to run")
	if err := prereq.NewInitializer(opts).Install(client); err != nil {
		return fmt.Errorf("prerequisites: %v", err)
	}

	clog.Printf("✅ installing kudo controller")
	if err := manager.NewInitializer(opts).Install(client); err != nil {
		return fmt.Errorf("manager: %v", err)
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
