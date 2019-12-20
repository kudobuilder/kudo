package generate

import (
	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
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
