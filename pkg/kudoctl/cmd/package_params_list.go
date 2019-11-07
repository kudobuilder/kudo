package cmd

import (
	"fmt"
	"io"
	"sort"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"

	"github.com/gosuri/uitable"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

type paramsListCmd struct {
	fs             afero.Fs
	out            io.Writer
	path           string
	descriptions   bool
	namesOnly      bool
	required       bool
	RepoName       string
	PackageVersion string
}

func newParamsListCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	list := &paramsListCmd{fs: fs, out: out}

	cmd := &cobra.Command{
		Use:     "list [operator]",
		Short:   "List operator parameters",
		Example: "  kubectl kudo package params list",
		RunE: func(cmd *cobra.Command, args []string) error {
			//list.home = Settings.Home
			if err := validateOperatorArg(args); err != nil {
				return err
			}
			list.path = args[0]
			return list.run(fs, &Settings)
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&list.descriptions, "descriptions", "d", false, "Display descriptions.")
	f.BoolVarP(&list.required, "required", "r", false, "Restricts list to params which have no defaults but are required.")
	f.BoolVar(&list.namesOnly, "names-only", false, "Display only names.")
	f.StringVar(&list.RepoName, "repo", "", "Name of repository configuration to use. (default defined by context)")
	f.StringVar(&list.PackageVersion, "version", "", "A specific package version on the official GitHub repo. (default to the most recent)")

	return cmd
}

// run provides a table listing the parameters.  There are 3 defined ways to view the table
// 1. names only using --names-only.  This is based on challenges with other approaches reading really long parameter names
// 2. name, default and required.  This is the **default**
// 3. name, default, required, desc.
func (c *paramsListCmd) run(fs afero.Fs, settings *env.Settings) error {

	repository, err := repo.ClientFromSettings(fs, settings.Home, c.RepoName)
	if err != nil {
		return errors.WithMessage(err, "could not build operator repository")
	}
	clog.V(4).Printf("repository used %s", repository)

	clog.V(3).Printf("getting package pkg files for %v with version: %v", c.path, c.PackageVersion)
	pf, err := kudo.PkgFiles(c.path, c.PackageVersion, repository)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve package files for operator: %s", c.path)
	}

	return displayParamsTable(pf, c)
}

func displayParamsTable(pf *packages.PackageFiles, cmd *paramsListCmd) error {
	sort.Sort(pf.Params)
	table := uitable.New()
	tValue := true
	// required
	if cmd.required {
		table.AddRow("Name")
		found := false
		for _, p := range pf.Params {
			if p.Default == nil && p.Required == &tValue {
				found = true
				table.AddRow(p.Name)
			}
		}
		if found {
			fmt.Fprintln(cmd.out, table)
		} else {
			fmt.Fprintf(cmd.out, "no required parameters without default values found\n")
		}
		return nil
	}
	// names only
	if cmd.namesOnly {
		table.AddRow("Name")
		for _, p := range pf.Params {
			table.AddRow(p.Name)
		}
		fmt.Fprintln(cmd.out, table)
		return nil
	}
	table.MaxColWidth = 35
	table.Wrap = true
	if cmd.descriptions {
		//table.MaxColWidth = 50
		table.AddRow("Name", "Default", "Required", "Descriptions")

	} else {
		table.AddRow("Name", "Default", "Required")
	}
	sort.Sort(pf.Params)
	for _, p := range pf.Params {
		pDefault := ""
		if p.Default != nil {
			pDefault = *p.Default
		}
		if cmd.descriptions {
			table.AddRow(p.Name, pDefault, *p.Required, p.Description)
		} else {
			table.AddRow(p.Name, pDefault, *p.Required)
		}
	}
	fmt.Fprintln(cmd.out, table)
	return nil
}