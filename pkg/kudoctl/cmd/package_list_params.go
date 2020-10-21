package cmd

import (
	"fmt"
	"io"
	"sort"

	"github.com/gosuri/uitable"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/generate"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/util/convert"
)

type packageListParamsCmd struct {
	fs              afero.Fs
	out             io.Writer
	pathOrName      string
	descriptions    bool
	namesOnly       bool
	requiredOnly    bool
	RepoName        string
	AppVersion      string
	OperatorVersion string
}

type Parameters []kudoapi.Parameter

// Len returns the number of params.
// This is needed to allow sorting of params.
func (p Parameters) Len() int { return len(p) }

// Swap swaps the position of two items in the params slice.
// This is needed to allow sorting of params.
func (p Parameters) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

// Less returns true if the name of a param a is less than the name of param b.
// This is needed to allow sorting of params.
func (p Parameters) Less(x, y int) bool {
	return p[x].Name < p[y].Name
}

const (
	pacakgeListParamsExample = `# show parameters from local-folder (where local-folder is a folder in the current directory)
  kubectl kudo package list parameters local-folder

  # show parameters from zookeeper (where zookeeper is name of package in KUDO repository)
  kubectl kudo package list parameters zookeeper`
)

func newPackageListParamsCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	list := &packageListParamsCmd{fs: fs, out: out}

	cmd := &cobra.Command{
		Use:     "parameters [operator]",
		Short:   "List operator parameters",
		Example: pacakgeListParamsExample,
		RunE: func(cmd *cobra.Command, args []string) error {

			path, patherr := generate.OperatorPath(fs)
			if patherr != nil {
				clog.V(2).Printf("operator path is not relative to execution")
			} else {
				list.pathOrName = path
			}
			err := validateOperatorArg(args)
			if err != nil && patherr != nil {
				return err
			}
			// if passed in... args take precedence
			if err == nil {
				list.pathOrName = args[0]
			}
			return list.run(&Settings)
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&list.descriptions, "descriptions", "d", false, "Display descriptions.")
	f.BoolVarP(&list.requiredOnly, "required", "r", false, "Show only parameters which have no defaults but are required.")
	f.BoolVar(&list.namesOnly, "names", false, "Display only names.")
	f.StringVar(&list.RepoName, "repo", "", "Name of repository configuration to use. (default defined by context)")
	f.StringVar(&list.AppVersion, "app-version", "", "A specific app version in the official GitHub repo. (default to the most recent)")
	f.StringVar(&list.OperatorVersion, "operator-version", "", "A specific operator version in the official GitHub repo. (default to the most recent)")

	return cmd
}

// run provides a table listing the parameters.  There are 3 defined ways to view the table
// 1. names only using --names.  This is based on challenges with other approaches reading really long parameter names
// 2. name, default and required.  This is the **default**
// 3. name, default, required, desc.
func (c *packageListParamsCmd) run(settings *env.Settings) error {
	if !onlyOneSet(c.requiredOnly, c.namesOnly, c.descriptions) {
		return fmt.Errorf("only one of the flags 'required', 'names', 'descriptions' can be set")
	}
	pr, err := packageDiscovery(c.fs, settings, c.RepoName, c.pathOrName, c.AppVersion, c.OperatorVersion)
	if err != nil {
		return err
	}

	if err := displayParamsTable(pr.OperatorVersion.Spec.Parameters, c.out, c.requiredOnly, c.namesOnly, c.descriptions); err != nil {
		return err
	}

	return nil
}

func displayParamsTable(params Parameters, out io.Writer, printRequired, printNames, printDesc bool) error {
	sort.Sort(params)
	table := uitable.New()
	tValue := true
	if printRequired {
		table.AddRow("Name")
		found := false
		for _, p := range params {
			if p.Default == nil && p.Required == &tValue {
				found = true
				table.AddRow(p.Name)
			}
		}
		if found {
			if _, err := fmt.Fprintln(out, table); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(out, "no required parameters without default values found\n"); err != nil {
				return err
			}
		}
	}
	if printNames {
		table.AddRow("Name")
		for _, p := range params {
			table.AddRow(p.Name)
		}
		if _, err := fmt.Fprintln(out, table); err != nil {
			return err
		}
	}
	table.MaxColWidth = 35
	table.Wrap = true
	if printDesc {
		table.AddRow("Name", "Default", "Required", "Immutable", "Descriptions")

	} else {
		table.AddRow("Name", "Default", "Required", "Immutable")
	}
	sort.Sort(params)
	for _, p := range params {
		if printDesc {
			table.AddRow(p.Name, convert.StringValue(p.Default), p.IsRequired(), p.IsImmutable(), p.Description)
		} else {
			table.AddRow(p.Name, convert.StringValue(p.Default), p.IsRequired(), p.IsImmutable())
		}
	}
	_, _ = fmt.Fprintln(out, table)
	return nil
}

func onlyOneSet(b bool, b2 bool, b3 bool) bool {
	// all false is ok all other combos need to verify only 1
	if !b && !b2 && !b3 {
		return true
	}
	return (b && !b2 && !b3) || (!b && b2 && !b3) || (!b && !b2 && b3)
}
