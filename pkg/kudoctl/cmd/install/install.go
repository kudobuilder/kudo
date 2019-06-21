package install

import (
	"fmt"
	"strings"

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
	Parameters      []string
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

	// Validate install parameters
	if err := validateInstallParameters(options.Parameters); err != nil {
		return errors.WithMessage(err, "could not parse parameters")
	}

	if err := installFrameworks(args, options); err != nil {
		return errors.WithMessage(err, "could not install framework(s)")
	}

	return nil
}

func validateInstallParameters(parameters []string) error {
	var errs []string

	for _, a := range parameters {
		// Using '=' as the delimiter. Split after the first delimiter to support using '=' in values
		s := strings.SplitN(a, "=", 2)
		if len(s) < 2 {
			errs = append(errs, fmt.Sprintf("parameter not set: %+v", a))
			continue
		}
		if s[0] == "" {
			errs = append(errs, fmt.Sprintf("parameter name can not be empty: %+v", a))
			continue
		}
		if s[1] == "" {
			errs = append(errs, fmt.Sprintf("parameter value can not be empty: %+v", a))
			continue
		}
	}

	if errs != nil {
		return errors.New(strings.Join(errs, ", "))
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

	// Downloading index.yaml file
	indexFile, err := r.DownloadIndexFile()
	if err != nil {
		return errors.WithMessage(err, "could not download index file")
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
		err := installFramework(name, "", *r, indexFile, kc, options)
		if err != nil {
			return err
		}
	}
	return nil
}

// installFramework is the umbrella for a single framework installation that gathers the business logic
// for a cluster and returns an error in case there is a problem
// TODO: needs testing
func installFramework(name, previous string, repository repo.FrameworkRepository, indexFile *repo.IndexFile, kc *kudo.Client, options *Options) error {

	var bundleVersion *repo.BundleVersion
	if options.PackageVersion == "" {
		bv, err := indexFile.GetByName(name)
		if err != nil {
			return errors.Wrapf(err, "getting %s in index file", name)
		}
		bundleVersion = bv
	} else {
		bv, err := indexFile.GetByNameAndVersion(name, options.PackageVersion)
		if err != nil {
			return errors.Wrapf(err, "getting %s in index file", name)
		}
		bundleVersion = bv
	}

	packageName := bundleVersion.Name + "-" + bundleVersion.Version

	crds, err := repository.GetPackage(packageName)
	if err != nil {
		return errors.Wrap(err, "failed to download bundle")
	}

	// Framework part

	// Check if Framework exists
	if !kc.FrameworkExistsInCluster(name, options.Namespace) {
		if err := installSingleFrameworkToCluster(name, options.Namespace, crds.Framework, kc); err != nil {
			return errors.Wrap(err, "installing single Framework")
		}
	}

	// FrameworkVersion part

	// Check if AnyFrameworkVersion for Framework exists
	if !kc.AnyFrameworkVersionExistsInCluster(name, options.Namespace) {
		// FrameworkVersion CRD for Framework does not exist
		if err := installSingleFrameworkVersionToCluster(name, options.Namespace, kc, crds.FrameworkVersion); err != nil {
			return errors.Wrapf(err, "installing FrameworkVersion CRD for framework %s", name)
		}
	}

	// Check if FrameworkVersion is out of sync with official FrameworkVersion for this Framework
	if !kc.FrameworkVersionInClusterOutOfSync(name, crds.FrameworkVersion.Spec.Version, options.Namespace) {
		// This happens when the given FrameworkVersion is not existing. E.g.
		// when a version has been installed that is not part of the official kudobuilder/frameworks repo.
		if !options.AutoApprove {
			fmt.Printf("No official FrameworkVersion has been found for \"%s\". "+
				"Do you want to install one? (Yes/no) ", name)
			if helpers.AskForConfirmation() {
				if err := installSingleFrameworkVersionToCluster(name, options.Namespace, kc, crds.FrameworkVersion); err != nil {
					return errors.Wrapf(err, "installing FrameworkVersion CRD for framework %s", name)
				}
			}
		} else {
			if err := installSingleFrameworkVersionToCluster(name, options.Namespace, kc, crds.FrameworkVersion); err != nil {
				return errors.Wrapf(err, "installing FrameworkVersion CRD for framework %s", name)
			}
		}

	}

	// Dependencies of the particular FrameworkVersion
	if options.AllDependencies {
		dependencyFrameworks, err := repository.GetFrameworkVersionDependencies(name, crds.FrameworkVersion)
		if err != nil {
			return errors.Wrap(err, "getting Framework dependencies")
		}
		for _, v := range dependencyFrameworks {
			// recursive function call
			// Dependencies should not be as big as that they will have an overflow in the function stack frame
			// installFramework makes sure that dependency Frameworks are created before the Framework itself
			// and it allows to inherit dependencies.
			if err := installFramework(v, name, repository, indexFile, kc, options); err != nil {
				return errors.Wrapf(err, "installing dependency Framework %s", v)
			}
		}
	}

	// Instances part
	// For a Framework without dependencies this means it creates the Instances object just after Framework and
	// FrameworkVersion objects are created to ensure Instances can be created.
	// This is also the part you end up when no dependencies are found or installed and all Framework and
	// FrameworkVersions are already installed.

	// Check if Instance exists in cluster
	// It won't create the Instance if any in combination with given Framework Name and FrameworkVersion exists
	if !kc.AnyInstanceExistsInCluster(name, options.Namespace, crds.FrameworkVersion.Spec.Version) {
		// This happens when the given FrameworkVersion is not existing. E.g.
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
func installSingleInstanceToCluster(name, previous string, instance *v1alpha1.Instance, kc *kudo.Client, options *Options) error {
	// Customizing Instance
	// TODO: traversing, e.g. check function that looksup if key exists in the current FrameworkVersion
	// That way just Parameters will be applied if they exist in the matching FrameworkVersion
	// More checking required
	// E.g. when installing with flag --all-dependencies to prevent overwriting dependency Instance name

	// This checks if flag --instance was set with a name and it is the not a dependency Instance
	if options.InstanceName != "" && previous == "" {
		instance.ObjectMeta.SetName(options.InstanceName)
	}
	if options.Parameters != nil {
		p := make(map[string]string)
		for _, a := range options.Parameters {
			s := strings.SplitN(a, "=", 2)
			p[s[0]] = s[1]
		}
		instance.Spec.Parameters = p
	}
	if _, err := kc.InstallInstanceObjToCluster(instance, options.Namespace); err != nil {
		return errors.Wrapf(err, "installing instance %s", name)
	}
	fmt.Printf("instance.%s/%s created\n", instance.APIVersion, instance.Name)
	return nil
}
