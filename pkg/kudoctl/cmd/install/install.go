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
		return errors.WithMessage(err, "could not install operator(s)")
	}

	return nil
}

// installFrameworks installs all frameworks specified as arguments into the cluster
func installFrameworks(args []string, options *Options) error {

	if len(args) < 1 {
		return fmt.Errorf("no argument provided")
	}

	if len(args) > 1 && options.PackageVersion != "" {
		return fmt.Errorf("--package-version not supported in multi operator install")
	}
	repoConfig := repo.Default

	// Initializing empty repo with given variables
	r, err := repo.NewFrameworkRepository(repoConfig)
	if err != nil {
		return errors.WithMessage(err, "could not build operator repository")
	}

	_, err = clientcmd.BuildConfigFromFlags("", options.KubeConfigPath)
	if err != nil {
		return errors.Wrap(err, "getting config failed")
	}

	kc, err := kudo.NewClient(options.Namespace, options.KubeConfigPath)
	if err != nil {
		return errors.Wrap(err, "creating kudo client")
	}

	for _, name := range args {
		err := installFramework(name, "", r, kc, options)
		if err != nil {
			return err
		}
	}
	return nil
}

// getPackageCRDs tries to look for package files resolving the operator name to:
// - a local tar.gz file
// - a local directory
// - a operator name in the remote repository
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

// installFramework is the umbrella for a single operator installation that gathers the business logic
// for a cluster and returns an error in case there is a problem
// TODO: needs testing
func installFramework(name, previous string, repository repo.Repository, kc *kudo.Client, options *Options) error {
	crds, err := getPackageCRDs(name, options, repository)
	if err != nil {
		return errors.Wrapf(err, "failed to install package: %s", name)
	}

	// Operator part

	// Check if Operator exists
	if !kc.OperatorExistsInCluster(name, options.Namespace) {
		if err := installSingleFrameworkToCluster(name, options.Namespace, crds.Operator, kc); err != nil {
			return errors.Wrap(err, "installing single Operator")
		}
	}

	// OperatorVersion part

	// Check if AnyFrameworkVersion for Operator exists
	if !kc.AnyOperatorVersionExistsInCluster(name, options.Namespace) {
		// OperatorVersion CRD for Operator does not exist
		if err := installSingleFrameworkVersionToCluster(name, options.Namespace, kc, crds.OperatorVersion); err != nil {
			return errors.Wrapf(err, "installing OperatorVersion CRD for operator: %s", name)
		}
	}

	// Check if OperatorVersion is out of sync with official OperatorVersion for this Operator
	if !kc.OperatorVersionInClusterOutOfSync(name, crds.OperatorVersion.Spec.Version, options.Namespace) {
		// This happens when the given OperatorVersion is not existing. E.g.
		// when a version has been installed that is not part of the official kudobuilder/frameworks repo.
		if !options.AutoApprove {
			fmt.Printf("No official OperatorVersion has been found for \"%s\". "+
				"Do you want to install one? (Yes/no) ", name)
			if helpers.AskForConfirmation() {
				if err := installSingleFrameworkVersionToCluster(name, options.Namespace, kc, crds.OperatorVersion); err != nil {
					return errors.Wrapf(err, "installing OperatorVersion CRD for operator %s", name)
				}
			}
		} else {
			if err := installSingleFrameworkVersionToCluster(name, options.Namespace, kc, crds.OperatorVersion); err != nil {
				return errors.Wrapf(err, "installing OperatorVersion CRD for operator %s", name)
			}
		}

	}

	// Dependencies of the particular OperatorVersion
	if options.AllDependencies {
		dependencyFrameworks, err := repo.GetFrameworkVersionDependencies(crds.OperatorVersion)
		if err != nil {
			return errors.Wrap(err, "getting Operator dependencies")
		}
		for _, v := range dependencyFrameworks {
			// recursive function call
			// Dependencies should not be as big as that they will have an overflow in the function stack frame
			// installFramework makes sure that dependency Operators are created before the Operator itself
			// and it allows to inherit dependencies.
			if err := installFramework(v, name, repository, kc, options); err != nil {
				return errors.Wrapf(err, "installing dependency Operator %s", v)
			}
		}
	}

	// Instances part
	// For a Operator without dependencies this means it creates the Instances object just after Operator and
	// OperatorVersion objects are created to ensure Instances can be created.
	// This is also the part you end up when no dependencies are found or installed and all Operator and
	// OperatorVersions are already installed.

	// Check if Instance exists in cluster
	// It won't create the Instance if any in combination with given Operator Name and OperatorVersion exists
	if !kc.AnyInstanceExistsInCluster(name, options.Namespace, crds.OperatorVersion.Spec.Version) {
		// This happens when the given OperatorVersion is not existing. E.g.
		// when a version has been installed that is not part of the official kudobuilder/frameworks repo.
		if !options.AutoApprove {
			fmt.Printf("No Instance tied to this \"%s\" version has been found. "+
				"Do you want to create one? (Yes/no) ", name)
			if helpers.AskForConfirmation() {
				// If Instance is a dependency we need to make sure installSingleInstanceToCluster is aware of it.
				// By having the previous string set we can make this distinction.
				if err := installSingleInstanceToCluster(name, previous, crds.Instance, kc, options); err != nil {
					return errors.Wrap(err, "installing single Instance")
				}
			}
		} else {
			if err := installSingleInstanceToCluster(name, previous, crds.Instance, kc, options); err != nil {
				return errors.Wrap(err, "installing single Instance")
			}
		}

	}
	return nil
}

// installSingleFrameworkToCluster installs a given Operator to the cluster
// TODO: needs testing
func installSingleFrameworkToCluster(name, namespace string, f *v1alpha1.Operator, kc *kudo.Client) error {
	if _, err := kc.InstallOperatorObjToCluster(f, namespace); err != nil {
		return errors.Wrapf(err, "installing %s-operator.yaml", name)
	}
	fmt.Printf("operator.%s/%s created\n", f.APIVersion, f.Name)
	return nil
}

// installSingleFrameworkVersionToCluster installs a given OperatorVersion to the cluster
// TODO: needs testing
func installSingleFrameworkVersionToCluster(name, namespace string, kc *kudo.Client, fv *v1alpha1.OperatorVersion) error {
	if _, err := kc.InstallOperatorVersionObjToCluster(fv, namespace); err != nil {
		return errors.Wrapf(err, "installing %s-operatorversion.yaml", name)
	}
	fmt.Printf("operatorversion.%s/%s created\n", fv.APIVersion, fv.Name)
	return nil
}

// installSingleInstanceToCluster installs a given Instance to the cluster
// TODO: needs more testing
func installSingleInstanceToCluster(name, previous string, instance *v1alpha1.Instance, kc *kudo.Client, options *Options) error {
	// Customizing Instance
	// TODO: traversing, e.g. check function that looksup if key exists in the current OperatorVersion
	// That way just Parameters will be applied if they exist in the matching OperatorVersion
	// More checking required
	// E.g. when installing with flag --all-dependencies to prevent overwriting dependency Instance name

	// This checks if flag --instance was set with a name and it is the not a dependency Instance
	if options.InstanceName != "" && previous == "" {
		instance.ObjectMeta.SetName(options.InstanceName)
	}
	if options.Parameters != nil {
		instance.Spec.Parameters = options.Parameters
	}
	if _, err := kc.InstallInstanceObjToCluster(instance, options.Namespace); err != nil {
		return errors.Wrapf(err, "installing instance %s", name)
	}
	fmt.Printf("instance.%s/%s created\n", instance.APIVersion, instance.Name)
	return nil
}
