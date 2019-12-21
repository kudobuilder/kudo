package cmd

import (
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/generate"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/prompt"
)

const (
	pkgAddTaskDesc = `Adds a task to existing operator package files.
`
	pkgAddTaskExample = `  kubectl kudo package add task
`
)

type packageAddTaskCmd struct {
	path        string
	interactive bool
	out         io.Writer
	fs          afero.Fs
}

// newPackageAddTaskCmd creates an operator tarball. fs is the file system, out is stdout for CLI
func newPackageAddTaskCmd(fs afero.Fs, out io.Writer) *cobra.Command {

	pkg := &packageAddTaskCmd{out: out, fs: fs}
	cmd := &cobra.Command{
		Use:     "task",
		Short:   "adds a task to the operator.yaml file",
		Long:    pkgAddTaskDesc,
		Example: pkgAddTaskExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := generate.OperatorPath(fs)
			if err != nil {
				return err
			}
			pkg.path = path
			if err := pkg.run(); err != nil {
				return err
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&pkg.interactive, "interactive", "i", false, "Interactive mode.")
	return cmd
}

func (pkg *packageAddTaskCmd) run() error {
	// interactive mode
	existing, err := generate.TaskList(pkg.fs, pkg.path)
	if err != nil {
		return err
	}
	task, err := prompt.ForTask(existing)
	if err != nil {
		return err
	}

	// ensure resources exist
	for _, resource := range task.Spec.Resources {
		err = generate.EnsureResource(pkg.fs, pkg.path, resource)
		if err != nil {
			return nil
		}
	}

	return generate.AddTask(pkg.fs, pkg.path, task)
}
