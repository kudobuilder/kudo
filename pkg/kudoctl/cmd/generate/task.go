package generate

import (
	"fmt"
	"path"

	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/reader"
)

func AddTask(fs afero.Fs, path string, task v1beta1.Task) error {
	p, err := reader.ReadDir(fs, path)
	if err != nil {
		return err
	}
	o := p.Files.Operator

	o.Tasks = append(o.Tasks, task)

	return writeOperator(fs, path, *o)
}

// TaskList provides a list of operator tasks
func TaskList(fs afero.Fs, path string) ([]v1beta1.Task, error) {
	p, err := reader.ReadDir(fs, path)
	if err != nil {
		return nil, err
	}

	return p.Files.Operator.Tasks, nil
}

func TaskInList(fs afero.Fs, path, taskName string) (bool, error) {
	list, err := TaskList(fs, path)
	if err != nil {
		return false, err
	}
	for _, task := range list {
		if task.Name == taskName {
			return true, nil
		}
	}
	return false, nil
}

// TaskKinds provides a list of task kinds what are supported via generators
func TaskKinds() []string {
	return []string{task.ApplyTaskKind, task.DeleteTaskKind} //, task.PipeTaskKind}
}

func AddResource(fs afero.Fs, dir string, resource string) error {

	// does "operator" path exist?  if not err
	exists, err := afero.DirExists(fs, dir)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("operator path %q does not exist", dir)
	}

	// does templates folder exist? if not mkdir
	templatePath := path.Join(dir, "templates")
	exists, err = afero.DirExists(fs, templatePath)
	if err != nil {
		return err
	}
	if !exists {
		err = fs.Mkdir(templatePath, 0755)
		if err != nil {
			return err
		}
	}

	// does resource exist? if not "touch" it, otherwise good
	resourcePath := path.Join(dir, "templates", resource)
	exists, err = afero.Exists(fs, resourcePath)
	if err != nil {
		return err
	}
	if !exists {
		err = afero.WriteFile(fs, resourcePath, []byte{}, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}
