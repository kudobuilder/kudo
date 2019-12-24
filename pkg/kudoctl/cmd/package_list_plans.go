package cmd

import (
	"fmt"
	"io"
	"sort"

	"github.com/gosuri/uitable"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

type packageListPlansCmd struct {
	fs             afero.Fs
	out            io.Writer
	path           string
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
			list.path = args[0]
			return list.run(fs, &Settings)
		},
	}

	f := cmd.Flags()
	f.StringVar(&list.RepoName, "repo", "", "Name of repository configuration to use. (default defined by context)")
	f.StringVar(&list.PackageVersion, "version", "", "A specific package version on the official GitHub repo. (default to the most recent)")

	return cmd
}

func (c *packageListPlansCmd) run(fs afero.Fs, settings *env.Settings) error {
	pf, err := packageDiscovery(fs, settings, c.RepoName, c.path, c.PackageVersion)
	if err != nil {
		return err
	}

	return displayPlanTable(pf.Files, c.out)
}

func displayPlanTable(pf *packages.Files, out io.Writer) error {
	sort.Sort(pf.Params.Parameters)
	table := uitable.New()
	table.AddRow("Name", "Phase", "Strategy", "Step", "Task")
	for name, plan := range pf.Operator.Plans {
		phase1 := true
		step1 := true
		for _, phase := range plan.Phases {
			for _, step := range phase.Steps {
				if phase1 && step1 {
					table.AddRow(name, phase.Name, plan.Strategy, step.Name, step.Tasks)
				}
				if !phase1 && step1 {
					table.AddRow("name", phase.Name, plan.Strategy, step.Name, step.Tasks)
				}
				if !phase1 && !step1 {
					table.AddRow("", "", "", step.Name, step.Tasks)
				}
				step1 = false
			}
			phase1 = false
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
