package generate

import (
	"errors"
	"fmt"
	"path"

	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

// OperatorCheck checks to see if operator generation makes sense
// fails if folder exits (non-destructive)
// if "operator.yaml" exists in current dir, we assume it's a mistake an error
func OperatorCheck(fs afero.Fs, dir string, overwrite bool) error {
	exists, err := afero.Exists(fs, dir)
	if err != nil {
		return err
	}
	if exists && !overwrite {
		return fmt.Errorf("folder %q already exists", dir)
	}

	exists, err = afero.Exists(fs, "operator.yaml")
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

	err := OperatorCheck(fs, dir, overwrite)
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
	fname := path.Join(dir, "operator.yaml")
	if err != nil {
		return err
	}

	// required empty settings
	op.Tasks = []v1beta1.Task{}
	op.Plans = make(map[string]v1beta1.Plan)

	o, err := yaml.Marshal(op)
	if err != nil {
		return err
	}

	err = afero.WriteFile(fs, fname, o, 0755)
	if err != nil {
		return err
	}

	return nil
}
