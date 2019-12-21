package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
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
	nameValid := func(input string) error {
		if len(input) < 1 {
			return errors.New("Task name must be > than 1 character")
		}
		exists, err := generate.TaskInList(pkg.fs, pkg.path, input)
		if err != nil {
			return err
		}
		if exists {
			return errors.New("Task name must be unique")
		}
		return nil
	}
	name, err := prompt.WithValidator("Task Name", "", nameValid)
	if err != nil {
		return err
	}

	kind, err := prompt.WithOptions("Task Kind", generate.TaskKinds(), false)
	if err != nil {
		return err
	}

	var again bool
	resources := []string{}
	for {
		resource, err := prompt.WithDefault("Task Resource", "")
		if err != nil {
			return err
		}
		resources = append(resources, ensureFileExtension(resource, "yaml"))

		again = prompt.Confirm("Add another resource")
		if !again {
			break
		}
	}

	for _, resource := range resources {
		err = generate.AddResource(pkg.fs, pkg.path, resource)
		if err != nil {
			return err
		}
	}

	//TODO (kensipe): lets add pipe tasks!
	spec := v1beta1.TaskSpec{
		ResourceTaskSpec: v1beta1.ResourceTaskSpec{Resources: resources},
	}

	task := v1beta1.Task{
		Name: name,
		Kind: kind,
		Spec: spec,
	}

	return generate.AddTask(pkg.fs, pkg.path, task)
}

func ensureFileExtension(fname, ext string) string {
	if strings.Contains(fname, ".") {
		return fname
	}
	return fmt.Sprintf("%s.%s", fname, ext)
}
