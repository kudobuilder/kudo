package cmd

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/generate"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/prompt"
)

const (
	pkgAddParameterDesc = `Adds a parameter to existing operator package files.
`
	pkgAddParameterExample = `  kubectl kudo package add parameter
`
)

type packageAddParameterCmd struct {
	path        string
	interactive bool
	out         io.Writer
	fs          afero.Fs
}

// newPackageAddParameterCmd creates an operator tarball. fs is the file system, out is stdout for CLI
func newPackageAddParameterCmd(fs afero.Fs, out io.Writer) *cobra.Command {

	pkg := &packageAddParameterCmd{out: out, fs: fs}
	cmd := &cobra.Command{
		Use:     "parameter",
		Short:   "adds a parameter to the params.yaml file",
		Long:    pkgAddParameterDesc,
		Example: pkgAddParameterExample,
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

// run returns the errors associated with cmd env
func (pkg *packageAddParameterCmd) run() error {
	// interactive mode
	nameValid := func(input string) error {
		if len(input) < 1 {
			return errors.New("Parameter name must be > than 1 character")
		}
		return nil
	}
	name, err := prompt.WithValidator("Parameter Name", "", nameValid)
	if err != nil {
		return err
	}

	value, err := prompt.WithDefault("Default Value", "")
	if err != nil {
		return err
	}

	displayName, err := prompt.WithDefault("Display Name", "")
	if err != nil {
		return err
	}

	// building param
	p := v1beta1.Parameter{
		DisplayName: displayName,
		Name:        name,
	}
	if value != "" {
		p.Default = &value
	}

	desc, err := prompt.WithDefault("Description", "")
	if err != nil {
		return err
	}
	if desc != "" {
		p.Description = desc
	}

	// order determines the default ("false" is preferred)
	requiredValues := []string{"false", "true"}
	required, err := prompt.WithOptions("Required", requiredValues)
	if err != nil {
		return err
	}
	if required == "true" {
		t := true
		p.Required = &t
	}

	//PlanNameList
	planNames, err := generate.PlanNameList(pkg.fs, pkg.path)
	if err != nil {
		return err
	}
	var trigger string
	fmt.Printf("testign")
	if len(planNames) == 0 {
		fmt.Printf("plans == 0")
		trigger, err = prompt.WithDefault("Trigger Plan", "")
	} else {
		fmt.Printf("names %v", planNames)
		trigger, err = prompt.WithOptions("Trigger Plan", planNames)
	}
	if err != nil {
		return err
	}
	if trigger != "" {
		p.Trigger = trigger
	}

	return generate.AddParameter(pkg.fs, pkg.path, &p)
}
