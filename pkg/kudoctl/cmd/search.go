package cmd

import (
	"errors"
	"fmt"
	"io"

	"github.com/gosuri/uitable"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
)

const searchDesc = `
This command searches the repository for a match on "contains" search criteria

Given an operator named foo-demo, a search for 'foo' or 'demo' would include this operator.
`

const searchExamples = `  kubectl kudo search foo
  kubectl kudo --repo community foo
`

type searchCmd struct {
	out         io.Writer
	fs          afero.Fs
	repoName    string
	home        kudohome.Home
	allVersions bool
	repoClient  *repo.Client
}

// newSearchCmd search for operator searches based on names
func newSearchCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	searchCmd := &searchCmd{fs: fs, out: out}

	cmd := &cobra.Command{
		Use:     "search [criteria]",
		Short:   "Search for operators in repository.",
		Long:    searchDesc,
		Example: searchExamples,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("this command must have only 1 search criterion")
			}
			searchCmd.home = Settings.Home
			return searchCmd.run(args[0])
		},
	}

	f := cmd.Flags()
	f.StringVar(&searchCmd.repoName, "repo", "", "Name of repository configuration to use. (default defined by context)")
	f.BoolVarP(&searchCmd.allVersions, "all-versions", "a", false, "Return all versions of found operators.")
	return cmd
}

// run initializes local config and installs KUDO manager to Kubernetes cluster.
func (s *searchCmd) run(criteria string) error {
	var err error
	if s.repoClient == nil {
		s.repoClient, err = repo.ClientFromSettings(fs, s.home, s.repoName)
		if err != nil {
			return err
		}
	}

	found, err := s.repoClient.Find(criteria, s.allVersions)
	if err != nil {
		return err
	}
	if len(found) == 0 {
		fmt.Fprint(s.out, "no operators found\n")
		return nil
	}
	table := uitable.New()
	table.AddRow("Name", "Operator Version", "App Version")
	preName := ""
	for _, operator := range found {
		if preName != operator.Name {
			table.AddRow(operator.Name, operator.OperatorVersion, operator.AppVersion)
			preName = operator.Name
		} else {
			table.AddRow("", operator.OperatorVersion, operator.AppVersion)
		}
	}
	fmt.Fprintln(s.out, table)
	return nil
}
