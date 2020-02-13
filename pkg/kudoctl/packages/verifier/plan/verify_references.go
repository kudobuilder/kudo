package plan

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/verifier"
)

// ReferenceVerifier verifies plans producing errors for plans referenced in param triggers that do not exist
// and warnings for missing mandatory plans.
type ReferenceVerifier struct{}

func (ReferenceVerifier) Verify(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	res.Merge(plansNotDefined(pf))
	res.Merge(hasMandatoryPlans(pf))

	return res
}

func hasMandatoryPlans(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	plans := pf.Operator.Plans

	// Currently only 'deploy' plan is mandatory
	if _, ok := plans[v1beta1.DeployPlanName]; !ok {
		res.AddErrors(fmt.Sprintf("an operator is required to have '%s' plan", v1beta1.DeployPlanName))
	}

	return res
}

func plansNotDefined(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	plans := pf.Operator.Plans

	for _, param := range pf.Params.Parameters {
		if param.Trigger != "" {
			if _, ok := plans[param.Trigger]; !ok {
				res.AddErrors(fmt.Sprintf("plan %q used in parameter %q is not defined", param.Trigger, param.Name))
			}
		}
	}

	return res
}
