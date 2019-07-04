package install

import (
	"fmt"
	"os"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/bundle"
	"github.com/kudobuilder/kudo/pkg/kudoctl/bundle/finder"
	"github.com/kudobuilder/kudo/pkg/kudoctl/http"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/check"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

// Options defines configuration options for the install command
type Options struct {
	InstanceName   string
	KubeConfigPath string
	Namespace      string
	Parameters     map[string]string
	PackageVersion string
	SkipInstance   bool
}

// DefaultOptions initializes the install command options to its defaults
var DefaultOptions = &Options{
	Namespace: "default",
}

// Run returns the errors associated with cmd env
func Run(cmd *cobra.Command, args []string, options *Options) error {

	err := validate(args, options)
	if err != nil {
		return err
	}

	err = installOperator(args[0], options)
	return err
}

func validate(args []string, options *Options) error {
	if len(args) != 1 {
		return fmt.Errorf("expecting exactly one argument - name of the package or path to install")
	}

	// If the $KUBECONFIG environment variable is set, use that
	if len(os.Getenv("KUBECONFIG")) > 0 {
		options.KubeConfigPath = os.Getenv("KUBECONFIG")
	}

	configPath, err := check.KubeConfigLocationOrDefault(options.KubeConfigPath)
	if err != nil {
		return fmt.Errorf("error when getting default kubeconfig path: %+v", err)
	}
	options.KubeConfigPath = configPath
	if err := check.ValidateKubeConfigPath(options.KubeConfigPath); err != nil {
		return errors.WithMessage(err, "could not check kubeconfig path")
	}
	_, err = clientcmd.BuildConfigFromFlags("", options.KubeConfigPath)
	if err != nil {
		return errors.Wrap(err, "getting config failed")
	}

	return nil
}

// getPackageCRDs tries to look for package files resolving the operator name to:
// - a local tar.gz file
// - a local directory
// - a url to a tar.gz
// - a operator name in the remote repository
// in that order. Should there exist a local folder e.g. `cassandra` it will take precedence
// over the remote repository package with the same name.
func getPackageCRDs(name string, options *Options, repository repo.Repository) (*bundle.PackageCRDs, error) {

	// Local files/folder have priority
	if _, err := os.Stat(name); err == nil {
		f, err := finder.NewLocal()
		if err != nil {
			return nil, err
		}
		b, err := f.GetBundle(name, options.PackageVersion)
		if err != nil {
			return nil, err
		}
		return b.GetCRDs()
	}

	if http.IsValidURL(name) {
		f, err := finder.NewURL()
		if err != nil {
			return nil, err
		}
		b, err := f.GetBundle(name, options.PackageVersion)
		if err != nil {
			return nil, err
		}
		return b.GetCRDs()
	}

	b, err := repository.GetBundle(name, options.PackageVersion)
	if err != nil {
		return nil, err
	}
	return b.GetCRDs()
}

// installOperator is the umbrella for a single operator installation that gathers the business logic
// for a cluster and returns an error in case there is a problem
// TODO: needs testing
func installOperator(operatorArgument string, options *Options) error {
	repository, err := repo.NewOperatorRepository(repo.Default)
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

	crds, err := getPackageCRDs(operatorArgument, options, repository)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve package CRDs for operator: %s", operatorArgument)
	}

	operatorName := crds.Operator.ObjectMeta.Name
	operatorVersion := crds.OperatorVersion.Spec.Version

	// Operator part

	// Check if Operator exists
	if !kc.OperatorExistsInCluster(crds.Operator.ObjectMeta.Name, options.Namespace) {
		if err := installSingleOperatorToCluster(operatorName, options.Namespace, crds.Operator, kc); err != nil {
			return errors.Wrap(err, "installing single Operator")
		}
	}

	// OperatorVersion part

	versionsInstalled, err := kc.OperatorVersionsInstalled(operatorName, options.Namespace)
	if err != nil {
		return errors.Wrap(err, "retrieving existing operator versions")
	}
	if !versionExists(versionsInstalled, operatorVersion) {
		// this version does not exist in the cluster
		if err := installSingleOperatorVersionToCluster(operatorName, options.Namespace, kc, crds.OperatorVersion); err != nil {
			return errors.Wrapf(err, "installing OperatorVersion CRD for operator: %s", operatorName)
		}
	}

	// Instances part
	// it creates the Instances object just after Operator and
	// OperatorVersion objects are created to ensure Instances can be created.

	// First make sure that our instance object is up to date with overrides from commandline
	applyInstanceOverrides(crds.Instance, options)

	// The user opted not to install the instance.
	if options.SkipInstance {
		return nil
	}

	// Check if Instance exists in cluster
	// It won't create the Instance if any in combination with given Operator Name, OperatorVersion and Instance OperatorName exists
	instanceName := crds.Instance.ObjectMeta.Name
	instanceExists, err := kc.InstanceExistsInCluster(operatorName, options.Namespace, crds.OperatorVersion.Spec.Version, instanceName)
	if err != nil {
		return errors.Wrapf(err, "verifying the instance does not already exist")
	}

	if !instanceExists {
		if err := installSingleInstanceToCluster(operatorName, crds.Instance, kc, options); err != nil {
			return errors.Wrap(err, "installing single instance")

		}

	} else {
		return fmt.Errorf("can not install instance '%s' of operator '%s-%s' because instance of that name already exists in namespace %s",
			instanceName, operatorName, crds.OperatorVersion.Spec.Version, options.Namespace)
	}
	return nil
}

func versionExists(versions []string, currentVersion string) bool {
	for _, v := range versions {
		if v == currentVersion {
			return true
		}
	}
	return false
}

// installSingleOperatorToCluster installs a given Operator to the cluster
// TODO: needs testing
func installSingleOperatorToCluster(name, namespace string, f *v1alpha1.Operator, kc *kudo.Client) error {
	if _, err := kc.InstallOperatorObjToCluster(f, namespace); err != nil {
		return errors.Wrapf(err, "installing %s-operator.yaml", name)
	}
	fmt.Printf("operator.%s/%s created\n", f.APIVersion, f.Name)
	return nil
}

// installSingleOperatorVersionToCluster installs a given OperatorVersion to the cluster
// TODO: needs testing
func installSingleOperatorVersionToCluster(name, namespace string, kc *kudo.Client, fv *v1alpha1.OperatorVersion) error {
	if _, err := kc.InstallOperatorVersionObjToCluster(fv, namespace); err != nil {
		return errors.Wrapf(err, "installing %s-operatorversion.yaml", name)
	}
	fmt.Printf("operatorversion.%s/%s created\n", fv.APIVersion, fv.Name)
	return nil
}

// installSingleInstanceToCluster installs a given Instance to the cluster
// TODO: needs more testing
func installSingleInstanceToCluster(name string, instance *v1alpha1.Instance, kc *kudo.Client, options *Options) error {
	// Customizing Instance
	// TODO: traversing, e.g. check function that looksup if key exists in the current OperatorVersion
	// That way just Parameters will be applied if they exist in the matching OperatorVersion

	if _, err := kc.InstallInstanceObjToCluster(instance, options.Namespace); err != nil {
		return errors.Wrapf(err, "installing instance %s", name)
	}
	fmt.Printf("instance.%s/%s created\n", instance.APIVersion, instance.Name)
	return nil
}

func applyInstanceOverrides(instance *v1alpha1.Instance, options *Options) {
	if options.InstanceName != "" {
		instance.ObjectMeta.SetName(options.InstanceName)
	}
	if options.Parameters != nil {
		instance.Spec.Parameters = options.Parameters
	}
}
