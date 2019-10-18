package install

import (
	"fmt"

	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	pkgresolver "github.com/kudobuilder/kudo/pkg/kudoctl/packages/resolver"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
)

// RepositoryOptions defines the options necessary for any cmd working with repository
type RepositoryOptions struct {
	RepoName string
}

// Options defines configuration options for the install command
type Options struct {
	RepositoryOptions
	InstanceName    string
	Parameters      map[string]string
	AppVersion      string
	OperatorVersion string
	SkipInstance    bool
	RequestTimeout  int64
	Wait            bool
}

// DefaultOptions initializes the install command options to its defaults
var DefaultOptions = &Options{}

// Run returns the errors associated with cmd env
func Run(args []string, options *Options, fs afero.Fs, settings *env.Settings) error {

	err := validate(args)
	if err != nil {
		return err
	}

	err = installOperator(args[0], options, fs, settings)
	return err
}

func validate(args []string) error {
	if len(args) != 1 {
		return clog.Errorf("expecting exactly one argument - name of the package or path to install")
	}

	return nil
}

// installOperator is installing single operator into cluster and returns error in case of error
func installOperator(operatorArgument string, options *Options, fs afero.Fs, settings *env.Settings) error {

	repository, err := repo.ClientFromSettings(fs, settings.Home, options.RepoName)
	if err != nil {
		return fmt.Errorf("could not build operator repository: %w", err)
	}
	clog.V(4).Printf("repository used %s", repository)

	kc, err := env.GetClient(settings)
	clog.V(3).Printf("acquiring kudo client")
	if err != nil {
		clog.V(3).Printf("failed to acquire client")
		return fmt.Errorf("creating kudo client: %w", err)
	}

	clog.V(3).Printf("getting operator package")

	resolver := pkgresolver.New(repository)
	pkg, err := resolver.Resolve(operatorArgument, options.AppVersion, options.OperatorVersion)
	if err != nil {
		return fmt.Errorf("failed to resolve operator package for: %s %w", operatorArgument, err)
	}

	return kudo.InstallPackage(kc, pkg.Resources, options.SkipInstance, options.InstanceName, settings.Namespace, options.Parameters, options.Wait)
}
