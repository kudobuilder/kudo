package generate

import (
	"github.com/spf13/afero"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

// AddMaintainer adds a maintainer to the operator.yaml
func AddMaintainer(fs afero.Fs, path string, m *kudoapi.Maintainer) error {

	pf, err := reader.PackageFilesFromDir(fs, path)
	if err != nil {
		return err
	}
	o := pf.Operator

	o.Maintainers = append(o.Maintainers, m)

	return writeOperator(fs, path, o)
}

// MaintainerList provides a list of operator maintainers
func MaintainerList(fs afero.Fs, path string) ([]*kudoapi.Maintainer, error) {
	pf, err := reader.PackageFilesFromDir(fs, path)
	if err != nil {
		return nil, err
	}

	return pf.Operator.Maintainers, nil
}
