package generate

import (
	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

// AddParameter writes a parameter to the params.yaml file
func AddParameter(fs afero.Fs, path string, p *packages.Parameter) error {

	pf, err := reader.PackageFilesFromDir(fs, path)
	if err != nil {
		return err
	}

	params := pf.Params
	params.Parameters = append(params.Parameters, *p)

	return writeParameters(fs, path, *params)
}

func ParameterNameList(fs afero.Fs, path string) (paramNames []string, err error) {
	pf, err := reader.PackageFilesFromDir(fs, path)
	if err != nil {
		return nil, err
	}

	for _, parameter := range pf.Params.Parameters {
		paramNames = append(paramNames, parameter.Name)
	}

	return paramNames, nil
}
