package generate

import (
	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

func AddMaintainer(fs afero.Fs, path string, m *v1beta1.Maintainer) error {

	p, err := reader.ReadDir(fs, path)
	if err != nil {
		return err
	}
	o := p.Files.Operator

	o.Maintainers = append(o.Maintainers, m)

	return writeOperator(fs, path, *o)
}
