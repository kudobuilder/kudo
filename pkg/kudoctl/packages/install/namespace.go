package install

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/engine/renderer"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

// installNamespace installs a namespace from package resources.
// If the resources contain a namespace manifest, it is rendered and
// applied to the namespace.
func installNamespace(
	client *kudo.Client,
	resources packages.Resources,
	parameters map[string]string) error {
	clog.V(3).Printf("creating namespace: %q", resources.Instance.Namespace)

	manifest := ""
	if resources.Operator.Spec.NamespaceManifest != "" {
		clog.V(3).Printf(
			"creating namespace with manifest named: %q",
			resources.Operator.Spec.NamespaceManifest)

		template, ok := resources.OperatorVersion.Spec.Templates[resources.Operator.Spec.NamespaceManifest]
		if !ok {
			return fmt.Errorf(
				"failed to find template for namespace manifest %q",
				resources.Operator.Spec.NamespaceManifest)
		}

		var err error
		manifest, err = renderNamespaceManifest(template, resources, parameters)
		if err != nil {
			return fmt.Errorf(
				"failed to render namespace manifest %q: %w",
				resources.Operator.Spec.NamespaceManifest,
				err)
		}
	}

	return client.CreateNamespace(resources.Instance.Namespace, manifest)
}

func renderNamespaceManifest(
	manifest string,
	resources packages.Resources,
	parameters map[string]string) (string, error) {
	configs := renderer.NewVariableMap().
		WithInstance("", resources.Instance.Name, resources.Instance.Namespace, "", "").
		WithResource(&resources).
		WithParameterStrings(parameters)
	engine := renderer.New()
	rendered, err := engine.Render("namespace", manifest, configs)
	if err != nil {
		return "", err
	}
	return rendered, nil
}
