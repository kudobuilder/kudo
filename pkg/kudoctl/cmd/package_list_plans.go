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

type packageListPlansCmd struct {
	fs             afero.Fs
	out            io.Writer
	pathOrName     string
	RepoName       string
	PackageVersion string
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

	return cmd
}

func (c *packageListPlansCmd) run(settings *env.Settings) error {
	pf, err := packageDiscovery(c.fs, settings, c.RepoName, c.pathOrName, c.PackageVersion)
	if err != nil {
		return err
	}

	return displayPlanTable(pf.Files, c.out)
}

func displayPlanTable(pf *packages.Files, out io.Writer) error {
	table := uitable.New()
	table.AddRow("Name", "Phase", "Strategy", "Step", "Task")
	for name, plan := range pf.Operator.Plans {
		var currentPlan, currentPhase, currentStep string
		for _, phase := range plan.Phases {
			for _, step := range phase.Steps {
				var planName, strategy string
				var phaseName, stepName string
				if name != currentPlan {
					planName = name
					strategy = string(plan.Strategy)
					currentPlan = name
				}
				if phase.Name != currentPhase {
					phaseName = phase.Name
					currentPhase = phase.Name
				}
				if step.Name != currentStep {
					stepName = step.Name
					currentStep = step.Name
				}
				table.AddRow(planName, phaseName, strategy, stepName, step.Tasks)
			}
		}
	}
	var err error
	if len(pf.Operator.Plans) == 0 {
		_, err = fmt.Fprintf(out, "no plans found\n")
	} else {
		_, err = fmt.Fprintln(out, table)
	}

	return err
}
