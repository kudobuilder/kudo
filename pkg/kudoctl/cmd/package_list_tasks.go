package cmd

import (
	"fmt"
	"io"

	"github.com/gosuri/uitable"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

type packageListTasksCmd struct {
	fs             afero.Fs
	out            io.Writer
	pathOrName     string
	RepoName       string
	PackageVersion string
}

const (
	packageListTasksExample = `# show tasks from local-folder (where local-folder is a folder in the current directory)
  kubectl kudo package list tasks local-folder

  # show tasks from zookeeper (where zookeeper is name of package in KUDO repository)
  kubectl kudo package list tasks zookeeper`
)

func newPackageListTasksCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	list := &packageListTasksCmd{fs: fs, out: out}

	cmd := &cobra.Command{
		Use:     "tasks [operator]",
		Short:   "List operator tasks",
		Example: packageListTasksExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOperatorArg(args); err != nil {
				return err
			}
			list.pathOrName = args[0]
			return list.run(&Settings)
		},
	}

	f := cmd.Flags()
	f.StringVar(&list.RepoName, "repo", "", "Name of repository configuration to use. (default defined by context)")
	f.StringVar(&list.PackageVersion, "version", "", "A specific package version on the official GitHub repo. (default to the most recent)")

	return cmd
}

// run provides a table listing the tasks for an operator.
func (c *packageListTasksCmd) run(settings *env.Settings) error {
	pf, err := packageDiscovery(c.fs, settings, c.RepoName, c.pathOrName, c.PackageVersion)
	if err != nil {
		return err
	}
	return displayTasksTable(pf.Files, c.out)
}

func displayTasksTable(pf *packages.Files, out io.Writer) error {
	table := uitable.New()
	table.AddRow("Name", "Kind", "Resources")
	for _, task := range pf.Operator.Tasks {
		if task.Kind == "Pipe" {
			table.AddRow(task.Name, task.Kind, task.Spec.Pod)
		} else {
			table.AddRow(task.Name, task.Kind, task.Spec.Resources)
		}
	}
	var err error
	if len(pf.Operator.Tasks) == 0 {
		_, err = fmt.Fprintf(out, "no tasks  found\n")
	} else {
		_, err = fmt.Fprintln(out, table)
	}
	return err
}
