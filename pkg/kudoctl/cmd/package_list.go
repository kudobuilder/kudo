package cmd

import (
	"fmt"
	"io"
	"os"

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

For list commands, the argument passed represents an operator.   That representation can be:

  - name of operator in the repository
  - url to the operator package (tgz file)
  - local operator package
  - local operator folder
`

const packageListExamples = `  # show list of parameters from local operator folder   
  kubectl kudo package list parameters ./local-folder

  # show list of parameters from zookeeper (where zookeeper is name of package in KUDO repository)
  kubectl kudo package list parameters zookeeper

  # show list of tasks from local operator folder
  kubectl kudo package list tasks ./local-folder

  # show list of tasks from zookeeper (where zookeeper is name of package in KUDO repository)
  kubectl kudo package list tasks zookeeper

  # show list of plans from local operator folder
  kubectl kudo package list plans ./local-folder

  # show plans from zookeeper (where zookeeper is name of package in KUDO repository)
  kubectl kudo package list plans zookeeper
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
func packageDiscovery(fs afero.Fs, settings *env.Settings, repoName, pathOrName, appVersion, operatorVersion string) (*packages.Resources, error) {
	repository, err := repo.ClientFromSettings(fs, settings.Home, repoName)
	if err != nil {
		return nil, fmt.Errorf("could not build operator repository: %w", err)
	}
	clog.V(3).Printf("repository used %s", repository)

	clog.V(3).Printf("getting package pkg files for %v with version: %v_%v", pathOrName, appVersion, operatorVersion)
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %v", err)
	}

	resolver := pkgresolver.NewPackageResolver(repository, wd)
	pr, err := resolver.Resolve(pathOrName, appVersion, operatorVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve package files for operator: %s: %w", pathOrName, err)
	}
	return pr.Resources, nil
}
