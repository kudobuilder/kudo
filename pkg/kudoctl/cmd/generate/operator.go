package generate

import (
	"errors"
	"fmt"
	"path"

	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

// CanGenerateOperator checks to see if operator generation makes sense (we don't generate over an operator or existing folder)
// fails if folder exits (non-destructive)
// if "operator.yaml" exists in current dir, we assume it's a mistake an error
func CanGenerateOperator(fs afero.Fs, dir string, overwrite bool) error {
	exists, err := afero.Exists(fs, dir)
	if err != nil {
		return err
	}
	if exists && !overwrite {
		return fmt.Errorf("folder %q already exists", dir)
	}

	exists, err = afero.Exists(fs, reader.OperatorFileName)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("operator.yaml exists in the current directory.  creating an operator in an operator is not supported")
	}
	return nil
}

// Operator generates an initial operator folder with a operator.yaml
func Operator(fs afero.Fs, dir string, op packages.OperatorFile, overwrite bool) error {
	err := CanGenerateOperator(fs, dir, overwrite)
	if err != nil {
		return err
	}

	exists, err := afero.DirExists(fs, dir)
	if err != nil {
		return err
	}

	if !exists {
		err = fs.Mkdir(dir, 0755)
	}
	if err != nil {
		return err
	}

	// required empty settings
	op.Tasks = []v1beta1.Task{}
	op.Plans = make(map[string]v1beta1.Plan)

	err = writeOperator(fs, dir, op)
	if err != nil {
		return err
	}

	pfname := path.Join(dir, reader.ParamsFileName)
	exists, err = afero.Exists(fs, pfname)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	// if params doesn't exist create it
	p := packages.ParamsFile{
		APIVersion: reader.APIVersion,
		Parameters: []v1beta1.Parameter{},
	}
	return writeParameters(fs, dir, p)
}

func writeParameters(fs afero.Fs, dir string, params packages.ParamsFile) error {
	p, err := yaml.Marshal(params)
	if err != nil {
		return err
	}

	fname := path.Join(dir, reader.ParamsFileName)
	return afero.WriteFile(fs, fname, p, 0755)
}

func writeOperator(fs afero.Fs, dir string, op packages.OperatorFile) error {
	o, err := yaml.Marshal(op)
	if err != nil {
		return err
	}

	fname := path.Join(dir, reader.OperatorFileName)
	return afero.WriteFile(fs, fname, o, 0755)
}
