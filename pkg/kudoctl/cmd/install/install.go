package install

import (
	"os"
	"strings"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/env"
	"github.com/kudobuilder/kudo/pkg/kudoctl/http"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/finder"
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

// GetPackageCRDs tries to look for package files resolving the operator name to:
// - a local tgz file
// - a local directory
// - a url to a tgz
// - an operator name in the remote repository
// in that order. Should there exist a local folder e.g. `cassandra` it will take precedence
// over the remote repository package with the same name.
func GetPackageCRDs(name string, version string, repository repo.Repository) (*packages.PackageCRDs, error) {

	// Local files/folder have priority
	if _, err := os.Stat(name); err == nil {
		clog.V(2).Printf("local operator discovered: %v", name)
		f := finder.NewLocal()
		b, err := f.GetPackage(name, version)
		if err != nil {
			return nil, err
		}
		return b.GetCRDs()
	}

	clog.V(3).Printf("no local operator discovered, looking for http")
	if http.IsValidURL(name) {
		clog.V(3).Printf("operator using http protocol for %v", name)
		f := finder.NewURL()
		b, err := f.GetPackage(name, version)
		if err != nil {
			return nil, err
		}
		return b.GetCRDs()
	}

	clog.V(3).Printf("no http discovered, looking for repository")
	b, err := repository.GetPackage(name, version)
	if err != nil {
		return nil, err
	}
	return b.GetCRDs()
}

// installOperator is installing single operator into cluster and returns error in case of error
func installOperator(operatorArgument string, options *Options, fs afero.Fs, settings *env.Settings) error {

	repository, err := repo.ClientFromSettings(fs, settings.Home, options.RepoName)
	if err != nil {
		return errors.WithMessage(err, "could not build operator repository")
	}
	clog.V(4).Printf("repository used %v", repository)

	kc, err := kudo.NewClient(settings.Namespace, settings.KubeConfig)
	clog.V(3).Printf("acquiring kudo client")
	if err != nil {
		clog.V(3).Printf("failed to acquire client")
		return errors.Wrap(err, "creating kudo client")
	}

	clog.V(3).Printf("getting package crds")
	crds, err := GetPackageCRDs(operatorArgument, options.PackageVersion, repository)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve package CRDs for operator: %s", operatorArgument)
	}

	return installCrds(crds, kc, options, settings)
}

func installCrds(crds *packages.PackageCRDs, kc *kudo.Client, options *Options, settings *env.Settings) error {
	// PRE-INSTALLATION SETUP
	operatorName := crds.Operator.ObjectMeta.Name
	clog.V(3).Printf("operator name: %v", operatorName)
	operatorVersion := crds.OperatorVersion.Spec.Version
	clog.V(3).Printf("operator version: %v", operatorVersion)
	// make sure that our instance object is up to date with overrides from commandline
	applyInstanceOverrides(crds.Instance, options)
	// this validation cannot be done earlier because we need to do it after applying things from commandline
	err := validateCrds(crds, options.SkipInstance)
	if err != nil {
		return err
	}

	// Operator part

	// Check if Operator exists
	if !kc.OperatorExistsInCluster(crds.Operator.ObjectMeta.Name, settings.Namespace) {
		if err := installSingleOperatorToCluster(operatorName, settings.Namespace, crds.Operator, kc); err != nil {
			return errors.Wrap(err, "installing single Operator")
		}
	}

	// OperatorVersion part
	versionsInstalled, err := kc.OperatorVersionsInstalled(operatorName, settings.Namespace)
	if err != nil {
		return errors.Wrap(err, "retrieving existing operator versions")
	}
	if !VersionExists(versionsInstalled, operatorVersion) {
		// this version does not exist in the cluster
		if err := installSingleOperatorVersionToCluster(operatorName, settings.Namespace, kc, crds.OperatorVersion); err != nil {
			return errors.Wrapf(err, "installing OperatorVersion CRD for operator: %s", operatorName)
		}
	}

	// Instances part
	// it creates the Instances object just after Operator and
	// OperatorVersion objects are created to ensure Instances can be created.

	// The user opted not to install the instance.
	if options.SkipInstance {
		return nil
	}

	// Check if Instance exists in cluster
	// It won't create the Instance if any in combination with given Operator Name, OperatorVersion and Instance OperatorName exists
	instanceName := crds.Instance.ObjectMeta.Name
	instanceExists, err := kc.InstanceExistsInCluster(operatorName, settings.Namespace, crds.OperatorVersion.Spec.Version, instanceName)
	if err != nil {
		return errors.Wrapf(err, "verifying the instance does not already exist")
	}

	if !instanceExists {
		if err := installSingleInstanceToCluster(operatorName, crds.Instance, kc, options, settings); err != nil {
			return errors.Wrap(err, "installing single instance")

		}

	} else {
		return clog.Errorf("can not install instance '%s' of operator '%s-%s' because instance of that name already exists in namespace %s",
			instanceName, operatorName, crds.OperatorVersion.Spec.Version, settings.Namespace)
	}
	return nil
}

func validateCrds(crds *packages.PackageCRDs, skipInstance bool) error {
	if skipInstance {
		// right now we are just validating parameters on instance, if we're not creating instance right now, there is nothing to validate
		clog.V(3).Printf("skipping instance...")
		return nil
	}
	parameters := crds.OperatorVersion.Spec.Parameters
	missingParameters := []string{}
	for _, p := range parameters {
		if p.Required && p.Default == nil {
			_, ok := crds.Instance.Spec.Parameters[p.Name]
			if !ok {
				missingParameters = append(missingParameters, p.Name)
			}
		}
	}

	if len(missingParameters) > 0 {
		return clog.Errorf("missing required parameters during installation: %s", strings.Join(missingParameters, ","))
	}
	return nil
}

// VersionExists looks for string version inside collection of versions
func VersionExists(versions []string, currentVersion string) bool {
	for _, v := range versions {
		if v == currentVersion {
			return true
		}
	}
	return false
}

// installSingleOperatorToCluster installs a given Operator to the cluster
// TODO: needs testing
func installSingleOperatorToCluster(name, namespace string, o *v1alpha1.Operator, kc *kudo.Client) error {
	if _, err := kc.InstallOperatorObjToCluster(o, namespace); err != nil {
		return errors.Wrapf(err, "installing %s-operator.yaml", name)
	}
	clog.V(2).Printf("operator.%s/%s created\n", o.APIVersion, o.Name)
	return nil
}

// installSingleOperatorVersionToCluster installs a given OperatorVersion to the cluster
// TODO: needs testing
func installSingleOperatorVersionToCluster(name, namespace string, kc *kudo.Client, ov *v1alpha1.OperatorVersion) error {
	if _, err := kc.InstallOperatorVersionObjToCluster(ov, namespace); err != nil {
		return errors.Wrapf(err, "installing %s-operatorversion.yaml", name)
	}
	clog.V(2).Printf("operatorversion.%s/%s created\n", ov.APIVersion, ov.Name)
	return nil
}

// installSingleInstanceToCluster installs a given Instance to the cluster
// TODO: needs more testing
func installSingleInstanceToCluster(name string, instance *v1alpha1.Instance, kc *kudo.Client, options *Options, settings *env.Settings) error {
	if _, err := kc.InstallInstanceObjToCluster(instance, settings.Namespace); err != nil {
		return errors.Wrapf(err, "installing instance %s", name)
	}
	clog.Printf("instance.%s/%s created\n", instance.APIVersion, instance.Name)
	return nil
}

func applyInstanceOverrides(instance *v1alpha1.Instance, options *Options) {
	if options.InstanceName != "" {
		instance.ObjectMeta.SetName(options.InstanceName)
		clog.V(3).Printf("instance name: %v", options.InstanceName)
	}
	if options.Parameters != nil {
		instance.Spec.Parameters = options.Parameters
		clog.V(3).Printf("parameters in use: %v", options.Parameters)
	}
}
