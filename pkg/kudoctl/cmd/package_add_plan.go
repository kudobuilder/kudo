package cmd

import (
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/generate"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/prompt"
)

const (
	pkgAddPlanDesc = `Adds a plan to existing operator package files.
`
	pkgAddPlanExample = `  kubectl kudo package add plan
`
)

type packageAddPlanCmd struct {
	path        string
	interactive bool
	out         io.Writer
	fs          afero.Fs
}

// newPackageAddPlanCmd adds a plan to an exist operator package
func newPackageAddPlanCmd(fs afero.Fs, out io.Writer) *cobra.Command {

	pkg := &packageAddPlanCmd{out: out, fs: fs}
	cmd := &cobra.Command{
		Use:     "plan",
		Short:   "adds a plan to the operator.yaml file",
		Long:    pkgAddPlanDesc,
		Example: pkgAddPlanExample,
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

func (pkg *packageAddPlanCmd) run() error {

	planNames, err := generate.PlanNameList(pkg.fs, pkg.path)
	if err != nil {
		return err
	}
	// get list of tasks
	tasks, err := generate.TaskList(pkg.fs, pkg.path)
	if err != nil {
		return err
	}

	// interactive mode
	planName, plan, err := prompt.ForPlan(planNames, tasks, pkg.fs, pkg.path, createTaskFromPrompts)
	if err != nil {
		return err
	}

	return generate.AddPlan(pkg.fs, pkg.path, planName, plan)
}
