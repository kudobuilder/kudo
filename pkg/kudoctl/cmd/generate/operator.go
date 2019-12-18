package generate

import (
	"errors"
	"fmt"
	"path"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"
)

// OperatorCheck checks to see if operator generation makes sense
// fails if folder exits (non-destructive)
// if "operator.yaml" exists in current dir, we assume it's a mistake an error
func OperatorCheck(fs afero.Fs, dir string) error {
	exists, err := afero.Exists(fs, dir)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("folder %q already exists", dir)
	}

	exists, err = afero.Exists(fs, "operator.yaml")
	if err != nil {
		return err
	}
	if exists {
		return errors.New("operator.yaml exists in the current directory.  creating an operator in an operator is not supported.")
	}
	return nil
}

// Operator generates an initial operator folder with a operator.yaml
func Operator(fs afero.Fs, dir string, op packages.OperatorFile) error {

	err := OperatorCheck(fs, dir)
	if err != nil {
		return err
	}

	err = fs.Mkdir(dir, 0755)
	if err != nil {
		return err
	}
	fname := path.Join(dir, "operator.yaml")
	if err != nil {
		return err
	}

	// required empty settings
	op.Tasks = []v1beta1.Task{}
	op.Plans = make(map[string]v1beta1.Plan)

	o, err := yaml.Marshal(op)

	err = afero.WriteFile(fs, fname, o, 0755)
	if err != nil {
		return err
	}

	return nil
}
