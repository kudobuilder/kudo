package cmd

import (
	"fmt"
	"io"

	"github.com/gosuri/uitable"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
)

type packageListTasksCmd struct {
	fs              afero.Fs
	out             io.Writer
	pathOrName      string
	RepoName        string
	AppVersion      string
	OperatorVersion string
}

const (
	packageListTasksExample = `# show tasks from local-folder (where local-folder is a folder in the current directory)
  kubectl kudo package list tasks local-folder

  # show tasks from zookeeper (where zookeeper is name of package in KUDO repository)
  kubectl kudo package list tasks zookeeper`
)

func newPackageListTasksCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	lc := &packageListTasksCmd{fs: fs, out: out}

	cmd := &cobra.Command{
		Use:     "tasks [operator]",
		Short:   "List operator tasks",
		Example: packageListTasksExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOperatorArg(args); err != nil {
				return err
			}
			lc.pathOrName = args[0]
			return lc.run(&Settings)
		},
	}

	f := cmd.Flags()
	f.StringVar(&lc.RepoName, "repo", "", "Name of repository configuration to use. (default defined by context)")
	f.StringVar(&lc.AppVersion, "app-version", "", "A specific app version in the official GitHub repo. (default to the most recent)")
	f.StringVar(&lc.OperatorVersion, "operator-version", "", "A specific operator version in the official GitHub repo. (default to the most recent)")

	return cmd
}

// run provides a table listing the tasks for an operator.
func (c *packageListTasksCmd) run(settings *env.Settings) error {
	pr, err := packageDiscovery(c.fs, settings, c.RepoName, c.pathOrName, c.AppVersion, c.OperatorVersion)
	if err != nil {
		return err
	}
	displayTasksTable(pr.OperatorVersion.Spec.Tasks, c.out)
	return nil
}

func displayTasksTable(tasks []kudoapi.Task, out io.Writer) {
	table := uitable.New()
	table.AddRow("Name", "Kind", "Resources")
	for _, task := range tasks {
		if task.Kind == "Pipe" {
			table.AddRow(task.Name, task.Kind, task.Spec.Pod)
		} else {
			table.AddRow(task.Name, task.Kind, task.Spec.Resources)
		}
	}
	if len(tasks) == 0 {
		fmt.Fprintf(out, "no tasks  found\n")
	} else {
		fmt.Fprintln(out, table)
	}
}
