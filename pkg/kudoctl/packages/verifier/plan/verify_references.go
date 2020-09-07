package plan

import (
	"fmt"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

var _ packages.Verifier = &ReferenceVerifier{}

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
	if _, ok := plans[kudoapi.DeployPlanName]; !ok {
		res.AddErrors(fmt.Sprintf("an operator is required to have '%s' plan", kudoapi.DeployPlanName))
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
