package generate

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"

	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
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
func Operator(fs afero.Fs, dir string, op *packages.OperatorFile, overwrite bool) error {
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
	op.Tasks = []kudoapi.Task{
		{
			Name: "deploy",
			Kind: task.ApplyTaskKind,
			Spec: kudoapi.TaskSpec{
				ResourceTaskSpec: kudoapi.ResourceTaskSpec{
					Resources: []string{},
				},
			},
		},
	}

	op.Plans = make(map[string]kudoapi.Plan)
	op.Plans["deploy"] = kudoapi.Plan{
		Strategy: "serial",
		Phases: []kudoapi.Phase{
			{
				Name:     "deploy",
				Strategy: "serial",
				Steps: []kudoapi.Step{
					{
						Name: "deploy",
						Tasks: []string{
							"deploy",
						},
					},
				},
			},
		},
	}

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
		Parameters: []packages.Parameter{},
	}
	err = writeParameters(fs, dir, p)
	if err != nil {
		return err
	}

	clog.V(0).Printf("Operator created. Use \n - package add parameter\n - package add plan\n - package add task\nor other package add commands to extend it.")

	return nil
}

func writeParameters(fs afero.Fs, dir string, params packages.ParamsFile) error {
	p, err := yaml.Marshal(params)
	if err != nil {
		return err
	}

	fname := path.Join(dir, reader.ParamsFileName)
	return afero.WriteFile(fs, fname, p, 0755)
}

func writeOperator(fs afero.Fs, dir string, op *packages.OperatorFile) error {
	o, err := yaml.Marshal(op)
	if err != nil {
		return err
	}

	fname := path.Join(dir, reader.OperatorFileName)
	return afero.WriteFile(fs, fname, o, 0755)
}

// OperatorPath determines the path to use as operator path for generators
// the path is either current "", or a dir with operator.yaml (if 1) else an error
// and is determined based on location of operator.yaml
func OperatorPath(fs afero.Fs) (string, error) {
	fname := "operator.yaml"

	exists, err := afero.Exists(fs, fname)
	if err != nil {
		return "", err
	}

	if exists {
		return "", nil
	}

	pat := path.Join("**", fname)
	// one more try
	files, err := afero.Glob(fs, pat)
	if err != nil {
		return "", err
	}
	if len(files) < 1 {
		return "", errors.New("no operator folder discovered")
	}
	if len(files) > 1 {
		return "", errors.New("multiple operator folders discovered")
	}
	return filepath.Dir(files[0]), nil
}
