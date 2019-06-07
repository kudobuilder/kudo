package install

import (
	"fmt"
	"strings"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/check"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/helpers"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

// CmdErrorProcessor returns the errors associated with cmd env
func CmdErrorProcessor(cmd *cobra.Command, args []string) error {

	// This makes --kubeconfig flag optional
	if _, err := cmd.Flags().GetString("kubeconfig"); err != nil {
		return fmt.Errorf("get flag: %+v", err)
	}

	if err := check.ValidateKubeConfigPath(); err != nil {
		return errors.WithMessage(err, "could not check kubeconfig path")
	}

	// Validate install parameters
	if err := validateInstallParameters(); err != nil {
		return errors.WithMessage(err, "could not parse parameters")
	}

	if err := verifyFrameworks(args); err != nil {
		return errors.WithMessage(err, "could not install framework(s)")
	}

	return nil
}

func validateInstallParameters() error {
	var errs []string

	if vars.Parameter != nil {

		for _, a := range vars.Parameter {
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
	}

	if errs != nil {
		return errors.New(strings.Join(errs, ", "))
	}

	return nil
}

func verifyFrameworks(args []string) error {

	if len(args) < 1 {
		return fmt.Errorf("no argument provided")
	}

	if len(args) > 1 && vars.PackageVersion != "" {
		return fmt.Errorf("--package-version not supported in multi framework install")
	}

	e := repo.Default

	// Initializing empty repo with given variables
	r, err := repo.NewFrameworkRepository(e)
	if err != nil {
		return errors.WithMessage(err, "could not build framework repository")
	}

	// Downloading index.yaml file
	indexFile, err := r.DownloadIndexFile()
	if err != nil {
		return errors.WithMessage(err, "could not download index file")
	}

	_, err = clientcmd.BuildConfigFromFlags("", vars.KubeConfigPath)
	if err != nil {
		return errors.Wrap(err, "getting config failed")
	}

	kc, err := kudo.NewKudoClient()
	if err != nil {
		return errors.Wrap(err, "creating kudo client")
	}

	for _, name := range args {
		err := verifySingleFramework(name, "", *r, indexFile, kc)
		if err != nil {
			return err
		}
	}
	return nil
}

// Todo: needs testing
// verifySingleFramework is the umbrella for a single framework installation that gathers the business logic
// for a cluster and returns an error in case there is a problem
func verifySingleFramework(name, previous string, repository repo.FrameworkRepository, indexFile *repo.IndexFile, kc *kudo.Client) error {

	var bundleVersion *repo.BundleVersion
	if vars.PackageVersion == "" {
		bv, err := indexFile.GetByName(name)
		if err != nil {
			return errors.Wrapf(err, "getting %s in index file", name)
		}
		bundleVersion = bv
	} else {
		bv, err := indexFile.GetByNameAndVersion(name, vars.PackageVersion)
		if err != nil {
			return errors.Wrapf(err, "getting %s in index file", name)
		}
		bundleVersion = bv
	}

	bundleName := bundleVersion.Name + "-" + bundleVersion.Version

	bundle, err := repository.DownloadBundle(bundleName)
	if err != nil {
		return errors.Wrap(err, "failed to download bundle")
	}

	// Framework part

	// Check if Framework exists
	if !kc.FrameworkExistsInCluster(name) {
		if err := installSingleFrameworkToCluster(name, bundle.Framework, kc); err != nil {
			return errors.Wrap(err, "installing single Framework")
		}
	}

	// FrameworkVersion part

	// Check if AnyFrameworkVersion for Framework exists
	if !kc.AnyFrameworkVersionExistsInCluster(name) {
		// FrameworkVersion CRD for Framework does not exist
		if err := installSingleFrameworkVersionToCluster(name, kc, bundle.FrameworkVersion); err != nil {
			return errors.Wrapf(err, "installing FrameworkVersion CRD for framework %s", name)
		}
	}

	// Check if FrameworkVersion is out of sync with official FrameworkVersion for this Framework
	if !kc.FrameworkVersionInClusterOutOfSync(name, bundle.FrameworkVersion.Spec.Version) {
		// This happens when the given FrameworkVersion is not existing. E.g.
		// when a version has been installed that is not part of the official kudobuilder/frameworks repo.
		if !vars.AutoApprove {
			fmt.Printf("No official FrameworkVersion has been found for \"%s\". "+
				"Do you want to install one? (Yes/no) ", name)
			if helpers.AskForConfirmation() {
				if err := installSingleFrameworkVersionToCluster(name, kc, bundle.FrameworkVersion); err != nil {
					return errors.Wrapf(err, "installing FrameworkVersion CRD for framework %s", name)
				}
			}
		} else {
			if err := installSingleFrameworkVersionToCluster(name, kc, bundle.FrameworkVersion); err != nil {
				return errors.Wrapf(err, "installing FrameworkVersion CRD for framework %s", name)
			}
		}

	}

	// Dependencies of the particular FrameworkVersion
	if vars.AllDependencies {
		dependencyFrameworks, err := repository.GetFrameworkVersionDependencies(name, bundle.FrameworkVersion)
		if err != nil {
			return errors.Wrap(err, "getting Framework dependencies")
		}
		for _, v := range dependencyFrameworks {
			// recursive function call
			// Dependencies should not be as big as that they will have an overflow in the function stack frame
			// verifySingleFramework makes sure that dependency Frameworks are created before the Framework itself
			// and it allows to inherit dependencies.
			if err := verifySingleFramework(v, name, repository, indexFile, kc); err != nil {
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
	if !kc.AnyInstanceExistsInCluster(name, bundle.FrameworkVersion.Spec.Version) {
		// This happens when the given FrameworkVersion is not existing. E.g.
		// when a version has been installed that is not part of the official kudobuilder/frameworks repo.
		if !vars.AutoApprove {
			fmt.Printf("No Instance tied to this \"%s\" version has been found. "+
				"Do you want to create one? (Yes/no) ", name)
			if helpers.AskForConfirmation() {
				// If Instance is a dependency we need to make sure installSingleInstanceToCluster is aware of it.
				// By having the previous string set we can make this distinction.
				if err := installSingleInstanceToCluster(name, previous, bundle.Instance, kc); err != nil {
					return errors.Wrap(err, "installing single Instance")
				}
			}
		} else {
			if err := installSingleInstanceToCluster(name, previous, bundle.Instance, kc); err != nil {
				return errors.Wrap(err, "installing single Instance")
			}
		}

	}
	return nil
}

// Todo: needs testing
// installSingleFrameworkToCluster installs a given Framework to the cluster
func installSingleFrameworkToCluster(name string, f *v1alpha1.Framework, kc *kudo.Client) error {
	if _, err := kc.InstallFrameworkObjToCluster(f); err != nil {
		return errors.Wrapf(err, "installing %s-framework.yaml", name)
	}
	fmt.Printf("framework.%s/%s created\n", f.APIVersion, f.Name)
	return nil
}

// Todo: needs testing
// installSingleFrameworkVersionToCluster installs a given FrameworkVersion to the cluster
func installSingleFrameworkVersionToCluster(name string, kc *kudo.Client, fv *v1alpha1.FrameworkVersion) error {
	if _, err := kc.InstallFrameworkVersionObjToCluster(fv); err != nil {
		return errors.Wrapf(err, "installing %s-frameworkversion.yaml", name)
	}
	fmt.Printf("frameworkversion.%s/%s created\n", fv.APIVersion, fv.Name)
	return nil
}

// Todo: needs more testing
// installSingleInstanceToCluster installs a given Instance to the cluster
func installSingleInstanceToCluster(name, previous string, instance *v1alpha1.Instance, kc *kudo.Client) error {
	// Customizing Instance
	// TODO: traversing, e.g. check function that looksup if key exists in the current FrameworkVersion
	// That way just Parameters will be applied if they exist in the matching FrameworkVersion
	// More checking required
	// E.g. when installing with flag --all-dependencies to prevent overwriting dependency Instance name

	// This checks if flag --instance was set with a name and it is the not a dependency Instance
	if vars.Instance != "" && previous == "" {
		instance.ObjectMeta.SetName(vars.Instance)
	}
	if vars.Parameter != nil {
		p := make(map[string]string)
		for _, a := range vars.Parameter {
			s := strings.SplitN(a, "=", 2)
			p[s[0]] = s[1]
		}
		instance.Spec.Parameters = p
	}
	if _, err := kc.InstallInstanceObjToCluster(instance); err != nil {
		return errors.Wrapf(err, "installing %s-instance.yaml", name)
	}
	fmt.Printf("instance.%s/%s created\n", instance.APIVersion, instance.Name)
	return nil
}
