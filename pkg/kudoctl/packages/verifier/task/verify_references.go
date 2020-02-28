package task

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

// ReferenceVerifier verifies tasks producing errors for tasks referenced in plans that do not exist and warnings for tasks which are not used in a plan
type ReferenceVerifier struct{}

func (ReferenceVerifier) Verify(pf *packages.Files) verifier.Result {
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
