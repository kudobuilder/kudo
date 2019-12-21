package generate

import (
	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

// AddParameter writes a parameter to the params.yaml file
func AddParameter(fs afero.Fs, path string, p *v1beta1.Parameter) error {

	pf, err := reader.ReadDir(fs, path)
	if err != nil {
		return err
	}

	params := pf.Files.Params
	params.Parameters = append(params.Parameters, *p)

	return writeParameters(fs, path, *params)
}
