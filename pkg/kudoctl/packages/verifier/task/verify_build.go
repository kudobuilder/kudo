package task

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

var _ packages.Verifier = &BuildVerifier{}

// BuildVerifier verifies tasks producing errors for tasks referenced in plans that do not exist and warnings for tasks which are not used in a plan
type BuildVerifier struct{}

func (BuildVerifier) Verify(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	res.Merge(tasksWellDefined(pf))
	return res
}

func tasksWellDefined(pf *packages.Files) verifier.Result {
	result := verifier.NewResult()
	for _, tt := range pf.Operator.Tasks {
		tt := tt

		switch tt.Kind {
		case task.KudoOperatorTaskKind:
			if tt.Spec.KudoOperatorTaskSpec.Package == "" {
				result.AddErrors(fmt.Sprintf("task validation error: kudo operator task '%s' has an empty package name", tt.Name))
			}
		default:
			if _, err := task.Build(&tt); err != nil {
				result.AddErrors(err.Error())
			}
		}
	}
	return result
}
