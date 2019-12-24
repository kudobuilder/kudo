package verify

import (
	"fmt"
	"strings"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/verifier"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/verifier/template"
)

var verifiers = []verifier.PackageVerifier{
	DuplicateVerifier{},
	InvalidCharVerifier{";,"},
	PlanVerifier{},
	template.ParametersVerifier{},
	template.ReferenceVerifier{},
}

// PackageFiles verifies operator package files
func PackageFiles(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, vv := range verifiers {
		res.Merge(vv.Verify(pf))
	}
	return res
}

// DuplicateVerifier provides verification that there are no duplicates disallowing casing (Kudo and kudo are duplicates)
type DuplicateVerifier struct{}

func (DuplicateVerifier) Verify(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	names := map[string]bool{}
	for _, param := range pf.Params.Parameters {
		name := strings.ToLower(param.Name)
		if names[name] {
			res.AddParamError(param, "has a duplicate")
		}
		names[name] = true
	}
	return res
}

type InvalidCharVerifier struct {
	InvalidChars string
}

func (v InvalidCharVerifier) Verify(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	for _, param := range pf.Params.Parameters {
		name := strings.ToLower(param.Name)
		for _, char := range name {
			if strings.Contains(v.InvalidChars, strings.ToLower(string(char))) {
				res.AddParamError(param, fmt.Sprintf("contains invalid character %q", char))
			}
		}

	}

	return res
}

// PlanVerifier verifies plans, with errors for step tasks that do not exist and warnings for tasks which are not used in a plan
type PlanVerifier struct{}

func (PlanVerifier) Verify(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	res.Merge(tasksNotDefined(pf))
	res.Merge(tasksDefinedNotUsed(pf))

	return res
}

func tasksDefinedNotUsed(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	usedTasks := make(map[string]bool)

	// map / set of all tasks that are referenced in a plan
	for _, plan := range pf.Operator.Plans {
		for _, phase := range plan.Phases {
			for _, step := range phase.Steps {
				for _, task := range step.Tasks {
					usedTasks[task] = true
				}
			}
		}
	}

	for _, task := range pf.Operator.Tasks {
		if _, ok := usedTasks[task.Name]; !ok {
			res.AddWarnings(fmt.Sprintf("task %q defined but not used", task.Name))
		}
	}

	return res
}

func tasksNotDefined(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	definedTasks := make(map[string]bool)

	// map of all tasks defined
	for _, task := range pf.Operator.Tasks {
		definedTasks[task.Name] = true
	}

	for planName, plan := range pf.Operator.Plans {
		for _, phase := range plan.Phases {
			for _, step := range phase.Steps {
				for _, task := range step.Tasks {
					if _, ok := definedTasks[task]; !ok {
						res.AddErrors(fmt.Sprintf("task %q in plan %q is not defined", task, planName))
					}
				}
			}
		}
	}

	return res
}
