package generate

import (
	"sort"

	"github.com/spf13/afero"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

// AddPlan adds a plan to the operator.yaml file
func AddPlan(fs afero.Fs, path string, planName string, plan *kudoapi.Plan) error {

	pf, err := reader.PackageFilesFromDir(fs, path)
	if err != nil {
		return err
	}

	o := pf.Operator
	if o.Plans == nil {
		o.Plans = make(map[string]kudoapi.Plan)
	}
	plans := o.Plans
	plans[planName] = *plan
	pf.Operator.Plans = plans

	return writeOperator(fs, path, o)
}

// PlanList provides a list of operator plans
func PlanList(fs afero.Fs, path string) (map[string]kudoapi.Plan, error) {
	p, err := reader.PackageFilesFromDir(fs, path)
	if err != nil {
		return nil, err
	}

	return p.Operator.Plans, nil
}

// PlanNameList provides a list of operator plan names
func PlanNameList(fs afero.Fs, path string) ([]string, error) {

	names := []string{}
	p, err := reader.PackageFilesFromDir(fs, path)
	if err != nil {
		return nil, err
	}
	for name := range p.Operator.Plans {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}
