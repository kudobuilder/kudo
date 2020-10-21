package cmd

import (
	"fmt"
	"io"
	"sort"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/thoas/go-funk"
	"github.com/xlab/treeprint"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
)

type packageListPlansCmd struct {
	fs                 afero.Fs
	out                io.Writer
	pathOrName         string
	RepoName           string
	AppVersion         string
	OperatorVersion    string
	WithTasksResources bool
}

const (
	packageListPlansExample = `  # show plans from local-folder (where local-folder is a folder in the current directory)
  kubectl kudo package list plans local-folder

  # show plans from zookeeper (where zookeeper is name of package in KUDO repository)
  kubectl kudo package list plans zookeeper`
)

func newPackageListPlansCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	lc := &packageListPlansCmd{fs: fs, out: out}

	cmd := &cobra.Command{
		Use:     "plans [operator]",
		Short:   "List operator plans",
		Example: packageListPlansExample,
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
	f.BoolVarP(&lc.WithTasksResources, "with-tasks", "t", false, "Display task resources with plans")

	return cmd
}

func (c *packageListPlansCmd) run(settings *env.Settings) error {
	pr, err := packageDiscovery(c.fs, settings, c.RepoName, c.pathOrName, c.AppVersion, c.OperatorVersion)
	if err != nil {
		return err
	}

	displayPlanTable(pr.OperatorVersion, c.WithTasksResources, c.out)
	return nil
}

func displayPlanTable(ov *kudoapi.OperatorVersion, withTasks bool, out io.Writer) {
	tree := treeprint.New()
	planNames := sortedPlanNames(ov.Spec.Plans)
	tree.SetValue("plans")
	for _, name := range planNames {
		plan := ov.Spec.Plans[name]
		pNode := tree.AddBranch(fmt.Sprintf("%s (%s)", name, plan.Strategy))

		for _, phase := range plan.Phases {
			phNode := pNode.AddMetaBranch("phase", fmt.Sprintf("%s (%s)", phase.Name, phase.Strategy))
			for _, step := range phase.Steps {
				sNode := phNode.AddMetaBranch("step", step.Name)
				for _, taskName := range step.Tasks {
					if withTasks {
						addTaskNodeWithResources(sNode, taskName, ov.Spec.Tasks)
					} else {
						sNode.AddNode(taskName)
					}
				}
			}
		}
	}

	if len(ov.Spec.Plans) == 0 {
		fmt.Fprintf(out, "no plans found\n")
	} else {
		fmt.Fprintln(out, tree.String())
	}
}

func sortedPlanNames(plans map[string]kudoapi.Plan) []string {
	planNames, ok := funk.Keys(plans).([]string)
	if !ok {
		panic("funk.Keys returned unexpected type")
	}
	sort.Strings(planNames)
	return planNames
}

func addTaskNodeWithResources(sNode treeprint.Tree, taskName string, tasks []kudoapi.Task) {
	for _, t := range tasks {
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
			case task.ToggleTaskKind:
				tNode := sNode.AddMetaBranch("toggle", taskName)
				tNode.AddMetaBranch("parameter", t.Spec.Parameter)
				for _, resource := range t.Spec.Resources {
					tNode.AddNode(resource)
				}
			case task.PipeTaskKind:
				tNode := sNode.AddMetaBranch("pipe", taskName)
				tNode.AddNode(t.Spec.Pod)
			case task.DummyTaskKind:
				sNode.AddMetaBranch("dummy", taskName)
			default:
				sNode.AddMetaBranch("unknown", taskName)
			}
		}
	}
}
