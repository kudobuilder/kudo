package plan

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/verifier"
)

// ReferenceVerifier verifies plans producing errors for plans referenced in param triggers that do not exist and warnings for plans which are not used in a param
type ReferenceVerifier struct{}

func (ReferenceVerifier) Verify(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	res.Merge(plansNotDefined(pf))
	res.Merge(plansDefinedNotUsed(pf))

	return res
}

func plansDefinedNotUsed(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	usedPlans := make(map[string]bool)

	// Mark reserved plan names as used
	for _, plan := range v1beta1.ReservedPlanNames {
		usedPlans[plan] = true
	}

	// Mark plans in param triggers as used
	for _, param := range pf.Params.Parameters {
		if param.Trigger != "" {
			usedPlans[param.Trigger] = true
		}
	}

	for name := range pf.Operator.Plans {
		if _, ok := usedPlans[name]; !ok {
			res.AddWarnings(fmt.Sprintf("plan %q defined but not used", name))
		}
	}

	return res
}

func plansNotDefined(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()
	plans := pf.Operator.Plans

	for _, param := range pf.Params.Parameters {
		if param.Trigger != "" {
			if _, ok := plans[param.Trigger]; !ok {
				res.AddErrors(fmt.Sprintf("plan %q used in param %q is not defined", param.Trigger, param.Name))
			}
		}
	}

	return res
}
