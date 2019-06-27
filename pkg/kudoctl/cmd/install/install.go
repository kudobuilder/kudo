package install

import (
	"fmt"
	"os"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/check"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/helpers"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

// Options defines configuration options for the install command
type Options struct {
	AllDependencies bool
	AutoApprove     bool
	InstanceName    string
	Namespace       string
	Parameters      map[string]string
	PackageVersion  string
	KubeConfigPath  string
}

// DefaultOptions initializes the install command options to its defaults
var DefaultOptions = &Options{
	Namespace: "default",
}

// Run returns the errors associated with cmd env
func Run(cmd *cobra.Command, args []string, options *Options) error {

	// This makes --kubeconfig flag optional
	if _, err := cmd.Flags().GetString("kubeconfig"); err != nil {
		return fmt.Errorf("get flag: %+v", err)
	}

	configPath, err := check.KubeConfigLocationOrDefault(options.KubeConfigPath)
	if err != nil {
		return fmt.Errorf("error when getting default kubeconfig path: %+v", err)
	}
	options.KubeConfigPath = configPath
	if err := check.ValidateKubeConfigPath(options.KubeConfigPath); err != nil {
		return errors.WithMessage(err, "could not check kubeconfig path")
	}

	if err := installFrameworks(args, options); err != nil {
		return errors.WithMessage(err, "could not install framework(s)")
	}

	return nil
}

// installFrameworks installs all frameworks specified as arguments into the cluster
func installFrameworks(args []string, options *Options) error {

	if len(args) < 1 {
		return fmt.Errorf("no argument provided")
	}

	if len(args) > 1 && options.PackageVersion != "" {
		return fmt.Errorf("--package-version not supported in multi framework install")
	}
	repoConfig := repo.Default

	// Initializing empty repo with given variables
	r, err := repo.NewFrameworkRepository(repoConfig)
	if err != nil {
		return errors.WithMessage(err, "could not build framework repository")
	}

	_, err = clientcmd.BuildConfigFromFlags("", options.KubeConfigPath)
	if err != nil {
		return errors.Wrap(err, "getting config failed")
	}

	kc, err := kudo.NewClient(options.Namespace, options.KubeConfigPath)
	if err != nil {
		return errors.Wrap(err, "creating kudo client")
	}

	for _, frameworkName := range args {
		err := installFramework(frameworkName, false, r, kc, options)
		if err != nil {
			return err
		}
	}
	return nil
}

// getPackageCRDs tries to look for package files resolving the framework name to:
// - a local tar.gz file
// - a local directory
// - a framework name in the remote repository
// in that order. Should there exist a local folder e.g. `cassandra` it will take precedence
// over the remote repository package with the same name.
func getPackageCRDs(name string, options *Options, repository repo.Repository) (*repo.PackageCRDs, error) {
	// Local files/folder have priority
	if _, err := os.Stat(name); err == nil {
		b, err := repo.NewBundle(name)
		if err != nil {
			return nil, err
		}
		return b.GetCRDs()
	}

	bundle, err := repository.GetPackageBundle(name, options.PackageVersion)
	if err != nil {
		return nil, err
	}
	return bundle.GetCRDs()
}

// installFramework is the umbrella for a single framework installation that gathers the business logic
// for a cluster and returns an error in case there is a problem
// TODO: needs testing
func installFramework(frameworkName string, isDependencyInstall bool, repository repo.Repository, kc *kudo.Client, options *Options) error {
	crds, err := getPackageCRDs(frameworkName, options, repository)
	if err != nil {
		return errors.Wrapf(err, "failed to install package: %s", frameworkName)
	}

	// Framework part

	// Check if Framework exists
	if !kc.FrameworkExistsInCluster(frameworkName, options.Namespace) {
		if err := installSingleFrameworkToCluster(frameworkName, options.Namespace, crds.Framework, kc); err != nil {
			return errors.Wrap(err, "installing single Framework")
		}
	}

	// FrameworkVersion part

	// Check if AnyFrameworkVersion for Framework exists
	if !kc.AnyFrameworkVersionExistsInCluster(frameworkName, options.Namespace) {
		// FrameworkVersion CRD for Framework does not exist
		if err := installSingleFrameworkVersionToCluster(frameworkName, options.Namespace, kc, crds.FrameworkVersion); err != nil {
			return errors.Wrapf(err, "installing FrameworkVersion CRD for framework: %s", frameworkName)
		}
	}

	// Check if FrameworkVersion is out of sync with official FrameworkVersion for this Framework
	if !kc.FrameworkVersionInClusterOutOfSync(frameworkName, crds.FrameworkVersion.Spec.Version, options.Namespace) {
		// This happens when the given FrameworkVersion is not existing. E.g.
		// when a version has been installed that is not part of the official kudobuilder/frameworks repo.
		if !options.AutoApprove {
			fmt.Printf("No official FrameworkVersion has been found for \"%s\". "+
				"Do you want to install one? (Yes/no) ", frameworkName)
			if helpers.AskForConfirmation() {
				if err := installSingleFrameworkVersionToCluster(frameworkName, options.Namespace, kc, crds.FrameworkVersion); err != nil {
					return errors.Wrapf(err, "installing FrameworkVersion CRD for framework %s", frameworkName)
				}
			}
		} else {
			if err := installSingleFrameworkVersionToCluster(frameworkName, options.Namespace, kc, crds.FrameworkVersion); err != nil {
				return errors.Wrapf(err, "installing FrameworkVersion CRD for framework %s", frameworkName)
			}
		}

	}

	// Dependencies of the particular FrameworkVersion
	if options.AllDependencies {
		dependencyFrameworks, err := repo.GetFrameworkVersionDependencies(crds.FrameworkVersion)
		if err != nil {
			return errors.Wrap(err, "getting Framework dependencies")
		}
		for _, v := range dependencyFrameworks {
			// recursive function call
			// Dependencies should not be as big as that they will have an overflow in the function stack frame
			// installFramework makes sure that dependency Frameworks are created before the Framework itself
			// and it allows to inherit dependencies.
			if err := installFramework(v, true, repository, kc, options); err != nil {
				return errors.Wrapf(err, "installing dependency Framework %s", v)
			}
		}
	}

	// Instances part
	// For a Framework without dependencies this means it creates the Instances object just after Framework and
	// FrameworkVersion objects are created to ensure Instances can be created.
	// This is also the part you end up when no dependencies are found or installed and all Framework and
	// FrameworkVersions are already installed.

	// First make sure that our instance object is up to date with overrides from commandline
	applyInstanceOverrides(crds.Instance, options, isDependencyInstall)

	// Check if Instance exists in cluster
	// It won't create the Instance if any in combination with given Framework Name, FrameworkVersion and Instance frameworkName exists
	instanceName := crds.Instance.ObjectMeta.Name
	instanceExists, err := kc.InstanceExistsInCluster(frameworkName, options.Namespace, crds.FrameworkVersion.Spec.Version, instanceName)
	if err != nil {
		return errors.Wrapf(err, "verifying the instance does not already exist")
	}

	if !instanceExists {
		// This happens when the given FrameworkVersion is not existing. E.g.
		// when a version has been installed that is not part of the official kudobuilder/frameworks repo.
		if !options.AutoApprove {
			fmt.Printf("No Instance tied to this \"%s\" version has been found. "+
				"Do you want to create one? (Yes/no) ", frameworkName)
			if helpers.AskForConfirmation() {
				// If Instance is a dependency we need to make sure installSingleInstanceToCluster is aware of it.
				// By having the previous string set we can make this distinction.
				if err := installSingleInstanceToCluster(frameworkName, crds.Instance, kc, options); err != nil {
					return errors.Wrap(err, "installing single Instance")
				}
			}
		} else {
			if err := installSingleInstanceToCluster(frameworkName, crds.Instance, kc, options); err != nil {
				return errors.Wrap(err, "installing single Instance")
			}
		}

	} else {
		return fmt.Errorf("can not install Instance %s of framework %s-%s because instance of that name already exists in namespace %s",
			instanceName, frameworkName, crds.FrameworkVersion.Spec.Version, options.Namespace)
	}
	return nil
}

// installSingleFrameworkToCluster installs a given Framework to the cluster
// TODO: needs testing
func installSingleFrameworkToCluster(name, namespace string, f *v1alpha1.Framework, kc *kudo.Client) error {
	if _, err := kc.InstallFrameworkObjToCluster(f, namespace); err != nil {
		return errors.Wrapf(err, "installing %s-framework.yaml", name)
	}
	fmt.Printf("framework.%s/%s created\n", f.APIVersion, f.Name)
	return nil
}

// installSingleFrameworkVersionToCluster installs a given FrameworkVersion to the cluster
// TODO: needs testing
func installSingleFrameworkVersionToCluster(name, namespace string, kc *kudo.Client, fv *v1alpha1.FrameworkVersion) error {
	if _, err := kc.InstallFrameworkVersionObjToCluster(fv, namespace); err != nil {
		return errors.Wrapf(err, "installing %s-frameworkversion.yaml", name)
	}
	fmt.Printf("frameworkversion.%s/%s created\n", fv.APIVersion, fv.Name)
	return nil
}

// installSingleInstanceToCluster installs a given Instance to the cluster
// TODO: needs more testing
func installSingleInstanceToCluster(name string, instance *v1alpha1.Instance, kc *kudo.Client, options *Options) error {
	// Customizing Instance
	// TODO: traversing, e.g. check function that looksup if key exists in the current FrameworkVersion
	// That way just Parameters will be applied if they exist in the matching FrameworkVersion

	if _, err := kc.InstallInstanceObjToCluster(instance, options.Namespace); err != nil {
		return errors.Wrapf(err, "installing instance %s", name)
	}
	fmt.Printf("instance.%s/%s created\n", instance.APIVersion, instance.Name)
	return nil
}

func applyInstanceOverrides(instance *v1alpha1.Instance, options *Options, isDependencyInstall bool) {
	// More checking required
	// E.g. when installing with flag --all-dependencies to prevent overwriting dependency Instance name
	// This checks if flag --instance was set with a name and it is the not a dependency Instance
	if options.InstanceName != "" && !isDependencyInstall {
		instance.ObjectMeta.SetName(options.InstanceName)
	}
	if options.Parameters != nil {
		instance.Spec.Parameters = options.Parameters
	}
}
