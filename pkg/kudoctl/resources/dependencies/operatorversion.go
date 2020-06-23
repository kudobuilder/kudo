package dependencies

import (
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	engtask "github.com/kudobuilder/kudo/pkg/engine/task"
)

// UpdateKudoOperatorTaskPackageNames sets the 'Package' and 'OperatorName'
// fields of the 'KudoOperatorTaskSpec' of an 'OperatorVersion' to the operator name
// initially referenced in the 'Package' field.
func UpdateKudoOperatorTaskPackageNames(
	dependencies []Dependency, operatorVersion *v1beta1.OperatorVersion) {
	tasks := operatorVersion.Spec.Tasks

	for i := range tasks {
		if tasks[i].Kind == engtask.KudoOperatorTaskKind {
			for _, dependency := range dependencies {
				if tasks[i].Spec.KudoOperatorTaskSpec.Package == dependency.PackageName {
					tasks[i].Spec.KudoOperatorTaskSpec.Package = dependency.Operator.Name
					break
				}
			}
		}
	}

	operatorVersion.Spec.Tasks = tasks
}
