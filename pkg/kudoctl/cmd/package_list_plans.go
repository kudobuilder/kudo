package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/xlab/treeprint"

	"github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

type packageListPlansCmd struct {
	fs                 afero.Fs
	out                io.Writer
	pathOrName         string
	RepoName           string
	PackageVersion     string
	WithTasksResources bool
}

const (
	packageListPlansExample = `# show plans from local-folder (where local-folder is a folder in the current directory)
  kubectl kudo package list plans local-folder

  # show plans from zookeeper (where zookeeper is name of package in KUDO repository)
  kubectl kudo package list plans zookeeper`
)

func newPackageListPlansCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	list := &packageListPlansCmd{fs: fs, out: out}

	cmd := &cobra.Command{
		Use:     "plans [operator]",
		Short:   "List operator plans",
		Example: packageListPlansExample,
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
	f.BoolVarP(&list.WithTasksResources, "with-tasks", "t", false, "Display task resources with plans")

	return cmd
}

func (c *packageListPlansCmd) run(settings *env.Settings) error {
	pf, err := packageDiscovery(c.fs, settings, c.RepoName, c.pathOrName, c.PackageVersion)
	if err != nil {
		return err
	}

	return displayPlanTable(pf.Files, c.WithTasksResources, c.out)
}

func displayPlanTable(pf *packages.Files, withTasks bool, out io.Writer) error {
	tree := treeprint.New()
	tree.SetValue("plans")
	for name, plan := range pf.Operator.Plans {
		pNode := tree.AddBranch(fmt.Sprintf("%s (%s)", name, plan.Strategy))

		for _, phase := range plan.Phases {
			phNode := pNode.AddMetaBranch("phase", fmt.Sprintf("%s (%s)", phase.Name, phase.Strategy))
			for _, step := range phase.Steps {
				sNode := phNode.AddMetaBranch("step", step.Name)
				for _, taskName := range step.Tasks {
					if withTasks {
						addTaskNodeWithResources(sNode, taskName, pf)
					} else {
						sNode.AddNode(taskName)
					}
				}
			}
		}
	}

	var err error
	if len(pf.Operator.Plans) == 0 {
		_, err = fmt.Fprintf(out, "no plans found\n")
	} else {
		_, err = fmt.Fprintln(out, tree.String())
	}

	return err
}

func addTaskNodeWithResources(sNode treeprint.Tree, taskName string, pf *packages.Files) {
	for _, t := range pf.Operator.Tasks {
		if t.Name == taskName {
			switch t.Kind {
			case task.ApplyTaskKind:
				tNode := sNode.AddMetaBranch("apply", taskName)
				for _, resource := range t.Spec.Resources {
					tNode.AddNode(resource)
				}
			case task.DeleteTaskKind:
				tNode := sNode.AddMetaBranch("delete", taskName)
				for _, resource := range t.Spec.Resources {
					tNode.AddNode(resource)
				}
			case task.PipeTaskKind:
				tNode := sNode.AddMetaBranch("pipe", taskName)
				tNode.AddNode(t.Spec.Pod)
			}
		}
	}
}
