package generate

import (
	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

func AddPlan(fs afero.Fs, path string, planName string, plan v1beta1.Plan) error {

	pf, err := reader.ReadDir(fs, path)
	if err != nil {
		return err
	}

	o := pf.Files.Operator
	plans := o.Plans
	plans[planName] = plan
	pf.Files.Operator.Plans = plans

	return writeOperator(fs, path, *o)
}

// PlanList provides a list of operator plans
func PlanList(fs afero.Fs, path string) (map[string]v1beta1.Plan, error) {
	p, err := reader.ReadDir(fs, path)
	if err != nil {
		return nil, err
	}

	return p.Files.Operator.Plans, nil
}

// PlanNameList provides a list of operator plan names
func PlanNameList(fs afero.Fs, path string) ([]string, error) {

	names := []string{}
	p, err := reader.ReadDir(fs, path)
	if err != nil {
		return nil, err
	}
	for name := range p.Files.Operator.Plans {
		names = append(names, name)
	}
	return names, nil
}
