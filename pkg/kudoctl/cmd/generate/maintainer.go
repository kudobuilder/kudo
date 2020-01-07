package generate

import (
	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

// AddMaintainer adds a maintainer to the operator.yaml
func AddMaintainer(fs afero.Fs, path string, m *v1beta1.Maintainer) error {

	p, err := reader.ReadDir(fs, path)
	if err != nil {
		return err
	}
	o := p.Files.Operator

	o.Maintainers = append(o.Maintainers, m)

	return writeOperator(fs, path, o)
}

// MaintainerList provides a list of operator maintainers
func MaintainerList(fs afero.Fs, path string) ([]*v1beta1.Maintainer, error) {
	p, err := reader.ReadDir(fs, path)
	if err != nil {
		return nil, err
	}

	return p.Files.Operator.Maintainers, nil
}
