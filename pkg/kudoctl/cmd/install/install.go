package install

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/resolver"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// RepositoryOptions defines the options necessary for any cmd working with repository
type RepositoryOptions struct {
	RepoName string
}

// Options defines configuration options for the install command
type Options struct {
	RepositoryOptions
	InstanceName   string
	Parameters     map[string]string
	PackageVersion string
	SkipInstance   bool
	RequestTimeout int64
}

// DefaultOptions initializes the install command options to its defaults
var DefaultOptions = &Options{}

// Run returns the errors associated with cmd env
func Run(args []string, options *Options, fs afero.Fs, settings *env.Settings) error {

	err := validate(args, options)
	if err != nil {
		return err
	}

	err = installOperator(args[0], options, fs, settings)
	return err
}

func validate(args []string, options *Options) error {
	if len(args) != 1 {
		return clog.Errorf("expecting exactly one argument - name of the package or path to install")
	}

	return nil
}

// installOperator is installing single operator into cluster and returns error in case of error
func installOperator(operatorArgument string, options *Options, fs afero.Fs, settings *env.Settings) error {

	repository, err := repo.ClientFromSettings(fs, settings.Home, options.RepoName)
	if err != nil {
		return errors.WithMessage(err, "could not build operator repository")
	}
	clog.V(4).Printf("repository used %s", repository)

	kc, err := env.GetClient(settings)
	clog.V(3).Printf("acquiring kudo client")
	if err != nil {
		clog.V(3).Printf("failed to acquire client")
		return errors.Wrap(err, "creating kudo client")
	}

	clog.V(3).Printf("getting package crds")

	pf := resolver.New(repository)
	pkg, err := pf.Resolve(operatorArgument, options.PackageVersion)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve package CRDs for operator: %s", operatorArgument)
	}

	return kudo.InstallPackage(kc, pkg.Resources, options.SkipInstance, options.InstanceName, settings.Namespace, options.Parameters)
}
