package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	pkgresolver "github.com/kudobuilder/kudo/pkg/kudoctl/packages/resolver"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
)

const packageListDesc = `
This command consists of multiple sub-commands to interact with KUDO packages.  These commands are used in the listing 
of an operator details such as parameters, tasks or plans.

List operator parameters
`

const packageListExamples = `  kubectl kudo package list parameters [operator folder]
  kubectl kudo package list task [operator folder]
  kubectl kudo package list plans [operator folder]
`

// newPackageParamsCmd for repo commands such as building a repo index
func newPackageParamsCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [FLAGS]",
		Short:   "list context from an operator package",
		Long:    packageListDesc,
		Example: packageListExamples,
	}
	cmd.AddCommand(newPackageListParamsCmd(fs, out))
	cmd.AddCommand(newPackageListPlansCmd(fs, out))
	cmd.AddCommand(newPackageListTasksCmd(fs, out))

	return cmd
}

// packageDiscovery is used by all list cmds to "discover" the packages
func packageDiscovery(fs afero.Fs, settings *env.Settings, repoName, path, packageVersion string) (*packages.Package, error) {
	repository, err := repo.ClientFromSettings(fs, settings.Home, repoName)
	if err != nil {
		return nil, fmt.Errorf("could not build operator repository: %w", err)
	}
	clog.V(4).Printf("repository used %s", repository)

	clog.V(3).Printf("getting package pkg files for %v with version: %v", path, packageVersion)
	resolver := pkgresolver.New(repository)
	pf, err := resolver.Resolve(path, packageVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve package files for operator: %s: %w", path, err)
	}
	return pf, nil
}
