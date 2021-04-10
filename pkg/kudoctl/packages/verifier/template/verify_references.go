package template

import (
	"fmt"

	engtask "github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

var _ packages.Verifier = &ReferenceVerifier{}

// ReferenceVerifier checks that all referenced templates exists (without errors)
// and warns if a template exists but isn't referenced in a plan
type ReferenceVerifier struct{}

func (ReferenceVerifier) Verify(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	templates := make(map[string]bool)
	for template := range pf.Templates {
		templates[template] = true
	}

	// conflated a bit...  the loop 1) confirms that all resources are defined templates, and 2) creates a map of all resources for next verification
	requiredTemplates := make(map[string]bool)
	for _, task := range pf.Operator.Tasks {
		var resources []string
		switch task.Kind {
		case engtask.ApplyTaskKind, engtask.DeleteTaskKind, engtask.ToggleTaskKind:
			resources = task.Spec.ResourceTaskSpec.Resources
		case engtask.PipeTaskKind:
			resources = append(resources, task.Spec.PipeTaskSpec.Pod)
		case engtask.KudoOperatorTaskKind:
			// param file is being stored in the templates folder, we need to make sure it's in here
			// to not treat it as error
			if task.Spec.KudoOperatorTaskSpec.ParameterFile != "" {
				resources = append(resources, task.Spec.KudoOperatorTaskSpec.ParameterFile)
			}
		default:
		}

		for _, r := range resources {
			requiredTemplates[r] = true
			if _, ok := templates[r]; !ok {
				res.AddErrors(fmt.Sprintf("template %q required by %s but is not defined", r, task.Name))
			}
		}
	}

	for template := range templates {
		// skip manifest file as it is already accounted for
		if template == pf.Operator.NamespaceManifest {
			continue
		}
		if _, ok := requiredTemplates[template]; !ok {
			res.AddWarnings(fmt.Sprintf("template %q is not referenced from any task", template))
		}
	}

	return res
}
