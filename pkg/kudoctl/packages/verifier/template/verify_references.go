package template

import (
	"fmt"

	engtask "github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/verifier"
)

var _ verifier.PackageVerifier = &ReferenceVerifier{}

// ReferenceVerifier checks that all referenced templates exists (without errors)
// and warns if a template exists but isn't referenced in a plan
type ReferenceVerifier struct{}

func (ReferenceVerifier) Verify(pf *packages.Files) (warnings verifier.Warnings, errors verifier.Errors) {
	templates := make(map[string]bool)
	for template := range pf.Templates {
		templates[template] = true
	}

	// conflated a bit...  the loop 1) confirms that all resources are defined templates, and 2) creates a map of all resources for next verification
	requiredTemplates := make(map[string]bool)
	for _, task := range pf.Operator.Tasks {
		var resources []string
		switch task.Kind {
		case engtask.ApplyTaskKind:
			resources = task.Spec.ResourceTaskSpec.Resources
		case engtask.DeleteTaskKind:
			resources = task.Spec.ResourceTaskSpec.Resources
		case engtask.PipeTaskKind:
			resources = append(resources, task.Spec.PipeTaskSpec.Pod)
		default:
		}

		for _, r := range resources {
			requiredTemplates[r] = true
			if _, ok := templates[r]; !ok {
				errors = append(errors, verifier.Error(fmt.Sprintf("template %q required by %s but is not defined", r, task.Name)))
			}
		}
	}

	for template := range templates {
		if _, ok := requiredTemplates[template]; !ok {
			warnings = append(warnings, verifier.Warning(fmt.Sprintf("template %q is not referenced from any task", template)))
		}
	}

	return warnings, errors
}
