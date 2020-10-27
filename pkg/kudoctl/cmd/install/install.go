package install

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/install"
	pkgresolver "github.com/kudobuilder/kudo/pkg/kudoctl/packages/resolver"
	deps "github.com/kudobuilder/kudo/pkg/kudoctl/resources/dependencies"
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
	WaitTime        int64
	CreateNameSpace bool
	InCluster       bool
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

func validate(args []string, opts *Options) error {
	if len(args) != 1 {
		return clog.Errorf("expecting exactly one argument - name of the package or path to install")
	}

	if opts.InCluster {
		if opts.SkipInstance || opts.RepoName != "" {
			return clog.Errorf("you can't use skip-instance or repo option when installing from in-cluster operators")
		}
	}
	return nil
}

// installOperator is installing single operator into cluster and returns error in case of error
func installOperator(operatorArgument string, options *Options, fs afero.Fs, settings *env.Settings) error {

	repoClient, err := repo.ClientFromSettings(fs, settings.Home, options.RepoName)
	if err != nil {
		return fmt.Errorf("could not build operator repository: %w", err)
	}
	clog.V(4).Printf("repository used %s", repoClient)

	kudoClient, err := env.GetClient(settings)
	clog.V(3).Printf("acquiring kudo client")
	if err != nil {
		clog.V(3).Printf("failed to acquire client")
		return fmt.Errorf("creating kudo client: %w", err)
	}

	clog.V(3).Printf("getting operator package")

	var resolver packages.Resolver
	if options.InCluster {
		resolver = pkgresolver.NewInClusterResolver(kudoClient, settings.Namespace)
	} else {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %v", err)
		}

		resolver = pkgresolver.NewPackageResolver(repoClient, wd)
	}

	pr, err := resolver.Resolve(operatorArgument, options.AppVersion, options.OperatorVersion)
	if err != nil {
		return fmt.Errorf("failed to resolve operator package for: %s %w", operatorArgument, err)
	}

	dependencies, err := deps.Resolve(pr.Resources.OperatorVersion, pr.DependenciesResolver)
	if err != nil {
		return err
	}

	installOpts := install.Options{
		SkipInstance:    options.SkipInstance,
		CreateNamespace: options.CreateNameSpace,
	}

	if options.Wait {
		waitDuration := time.Duration(options.WaitTime) * time.Second
		installOpts.Wait = &waitDuration
	}

	return install.Package(
		kudoClient,
		options.InstanceName,
		settings.Namespace,
		*pr.Resources,
		options.Parameters,
		dependencies,
		installOpts)
}
