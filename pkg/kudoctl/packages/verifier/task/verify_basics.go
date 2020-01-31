package task

import (
	"github.com/kudobuilder/kudo/pkg/engine/task"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/verifier"
)

// ReferenceVerifier verifies tasks producing errors for tasks referenced in plans that do not exist and warnings for tasks which are not used in a plan
type BasicVerifier struct{}

func (BasicVerifier) Verify(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	res.Merge(tasksWellDefined(pf))
	return res
}

func tasksWellDefined(pf *packages.Files) verifier.Result {
	result := verifier.NewResult()
	for _, tt := range pf.Operator.Tasks {
		tt := tt

		if _, err := task.Build(&tt); err != nil {
			result.AddErrors(err.Error())
		}
	}
	return result
}
