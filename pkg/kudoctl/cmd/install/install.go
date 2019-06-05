package install

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/kudoctl/util/check"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/helpers"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	bundlePath     = "%s/%s"
	fwPath         = "%s/%s-framework.yaml"
	fwVersionPath  = "%s/%s-frameworkversion.yaml"
	fwInstancePath = "%s/%s-instance.yaml"
)

// CmdErrorProcessor returns the errors associated with cmd env
func CmdErrorProcessor(cmd *cobra.Command, args []string) error {

	// default repo hardcoded for 0.2.0
	// this won't work on windows
	vars.RepoPath = ".kudo/repository"

	// This makes --kubeconfig flag optional
	if _, err := cmd.Flags().GetString("kubeconfig"); err != nil {
		return fmt.Errorf("get flag: %+v", err)
	}

	if err := check.ValidateKubeConfigPath(); err != nil {
		return errors.WithMessage(err, "could not check kubeconfig path")
	}

	// Checking repo path of provided parameter or env variable, if none provided looking up default directory
	if err := check.RepoPath(); err != nil {
		if err = helpers.CreateRepoPath(); err != nil {
			return errors.WithMessage(err, "could not create repo path")
		}
	}

	if err := verifyFrameworks(args); err != nil {
		return errors.WithMessage(err, "could not install framework(s)")
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

	e := repo.NewRepositoryConfiguration()

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
func verifySingleFramework(name, previous string, r repo.FrameworkRepository, i *repo.IndexFile, kc *kudo.Client) error {

	var bundleVersion *repo.BundleVersion
	if vars.PackageVersion == "" {
		bv, err := i.GetByName(name)
		if err != nil {
			return errors.Wrapf(err, "getting %s in index file", name)
		}
		bundleVersion = bv
	} else {
		bv, err := i.GetByNameAndVersion(name, vars.PackageVersion)
		if err != nil {
			return errors.Wrapf(err, "getting %s in index file", name)
		}
		bundleVersion = bv
	}

	// checking if bundle exists locally already
	bundleName := bundleVersion.Name + "-" + bundleVersion.Version
	bundlePath := fmt.Sprintf(bundlePath, vars.RepoPath, bundleName)

	if _, err := os.Stat(bundlePath); err != nil && os.IsNotExist(err) {
		if err = r.DownloadBundleFile(bundleName); err != nil {
			return errors.Wrap(err, "failed to download bundle")
		}
	}

	// Framework part

	// Check if Framework exists
	if !kc.FrameworkExistsInCluster(name) {
		if err := installSingleFrameworkToCluster(name, bundlePath, kc); err != nil {
			return errors.Wrap(err, "installing single Framework")
		}
	}

	// FrameworkVersion part

	// Get the string of the version in FrameworkVersion of a selected Framework
	frameworkVersion, err := r.GetFrameworkVersion(name, bundlePath)
	if err != nil {
		return errors.Wrap(err, "getting FrameworkVersion version")
	}

	// Check if AnyFrameworkVersion for Framework exists
	if !kc.AnyFrameworkVersionExistsInCluster(name) {
		// FrameworkVersion CRD for Framework does not exist
		if err := installSingleFrameworkVersionToCluster(name, bundlePath, kc); err != nil {
			return errors.Wrap(err, "installing single FrameworkVersion")
		}
	}

	// Check if FrameworkVersion is out of sync with official FrameworkVersion for this Framework
	if !kc.FrameworkVersionInClusterOutOfSync(name, frameworkVersion) {
		// This happens when the given FrameworkVersion is not existing. E.g.
		// when a version has been installed that is not part of the official kudobuilder/frameworks repo.
		if !vars.AutoApprove {
			fmt.Printf("No official FrameworkVersion has been found for \"%s\". "+
				"Do you want to install one? (Yes/no) ", name)
			if helpers.AskForConfirmation() {
				if err := installSingleFrameworkVersionToCluster(name, bundlePath, kc); err != nil {
					return errors.Wrapf(err, "installing single FrameworkVersion %s", name)
				}
			}
		} else {
			if err := installSingleFrameworkVersionToCluster(name, bundlePath, kc); err != nil {
				return errors.Wrapf(err, "installing single FrameworkVersion %s", name)
			}
		}

	}

	// Dependencies of the particular FrameworkVersion
	if vars.AllDependencies {
		dependencyFrameworks, err := r.GetFrameworkVersionDependencies(name, bundlePath)
		if err != nil {
			return errors.Wrap(err, "getting Framework dependencies")
		}
		for _, v := range dependencyFrameworks {
			// recursive function call
			// Dependencies should not be as big as that they will have an overflow in the function stack frame
			// verifySingleFramework makes sure that dependency Frameworks are created before the Framework itself
			// and it allows to inherit dependencies.
			if err := verifySingleFramework(v, name, r, i, kc); err != nil {
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
	if !kc.AnyInstanceExistsInCluster(name, frameworkVersion) {
		// This happens when the given FrameworkVersion is not existing. E.g.
		// when a version has been installed that is not part of the official kudobuilder/frameworks repo.
		if !vars.AutoApprove {
			fmt.Printf("No Instance tied to this \"%s\" version has been found. "+
				"Do you want to create one? (Yes/no) ", name)
			if helpers.AskForConfirmation() {
				// If Instance is a dependency we need to make sure installSingleInstanceToCluster is aware of it.
				// By having the previous string set we can make this distinction.
				if err := installSingleInstanceToCluster(name, previous, bundlePath, kc); err != nil {
					return errors.Wrap(err, "installing single Instance")
				}
			}
		} else {
			if err := installSingleInstanceToCluster(name, previous, bundlePath, kc); err != nil {
				return errors.Wrap(err, "installing single Instance")
			}
		}

	}
	return nil
}

// Todo: needs testing
// installSingleFrameworkToCluster installs a given Framework to the cluster
func installSingleFrameworkToCluster(name, path string, kc *kudo.Client) error {
	frameworkPath := fmt.Sprintf(fwPath, path, name)
	frameworkYamlFile, err := os.Open(frameworkPath)
	if err != nil {
		return errors.Wrap(err, "failed opening framework file")
	}

	frameworkByteValue, err := ioutil.ReadAll(frameworkYamlFile)
	if err != nil {
		return errors.Wrap(err, "failed reading framework file")
	}

	var f v1alpha1.Framework
	if err = yaml.Unmarshal(frameworkByteValue, &f); err != nil {
		return errors.Wrapf(err, "unmarshalling %s-framework.yaml content", name)
	}

	if _, err = kc.InstallFrameworkObjToCluster(&f); err != nil {
		return errors.Wrapf(err, "installing %s-framework.yaml", name)
	}
	fmt.Printf("framework.%s/%s created\n", f.APIVersion, f.Name)
	return nil
}

// Todo: needs testing
// installSingleFrameworkVersionToCluster installs a given FrameworkVersion to the cluster
func installSingleFrameworkVersionToCluster(name, path string, kc *kudo.Client) error {
	frameworkVersionPath := fmt.Sprintf(fwVersionPath, path, name)
	frameworkVersionYamlFile, err := os.Open(frameworkVersionPath)
	if err != nil {
		return errors.Wrap(err, "failed opening frameworkversion file")
	}

	frameworkVersionByteValue, err := ioutil.ReadAll(frameworkVersionYamlFile)
	if err != nil {
		return errors.Wrap(err, "failed reading frameworkversion file")
	}

	var fv v1alpha1.FrameworkVersion
	if err = yaml.Unmarshal(frameworkVersionByteValue, &fv); err != nil {
		return errors.Wrapf(err, "unmarshalling %s-frameworkversion.yaml content", name)
	}

	if _, err = kc.InstallFrameworkVersionObjToCluster(&fv); err != nil {
		return errors.Wrapf(err, "installing %s-frameworkversion.yaml", name)
	}
	fmt.Printf("frameworkversion.%s/%s created\n", fv.APIVersion, fv.Name)
	return nil
}

// Todo: needs more testing
// installSingleInstanceToCluster installs a given Instance to the cluster
func installSingleInstanceToCluster(name, previous, path string, kc *kudo.Client) error {
	frameworkInstancePath := fmt.Sprintf(fwInstancePath, path, name)
	frameworkInstanceYamlFile, err := os.Open(frameworkInstancePath)
	if err != nil {
		return errors.Wrap(err, "failed opening instance file")
	}

	frameworkInstanceByteValue, err := ioutil.ReadAll(frameworkInstanceYamlFile)
	if err != nil {
		return errors.Wrap(err, "failed reading instance file")
	}

	var i v1alpha1.Instance
	if err = yaml.Unmarshal(frameworkInstanceByteValue, &i); err != nil {
		return errors.Wrapf(err, "unmarshalling %s-instance.yaml content", name)
	}
	// Customizing Instance
	// TODO: traversing, e.g. check function that looksup if key exists in the current FrameworkVersion
	// That way just Parameters will be applied if they exist in the matching FrameworkVersion
	// More checking required
	// E.g. when installing with flag --all-dependencies to prevent overwriting dependency Instance name

	// This checks if flag --instance was set with a name and it is the not a dependency Instance
	if vars.Instance != "" && previous == "" {
		i.ObjectMeta.SetName(vars.Instance)
	}
	if vars.Parameter != nil {
		p := make(map[string]string)
		for _, a := range vars.Parameter {
			// Using similar to CSV "," as the delimiter for now
			// Split just after the first delimiter to support e.g. zk-zk-0.zk-hs:2181,zk-zk-1.zk-hs:2181 as value
			s := strings.SplitN(a, ",", 2)
			if len(s) < 2 {
				return fmt.Errorf("parameter not set: %+v", s)
			}
			if s[0] == "" {
				return fmt.Errorf("parameter can not be empty: %+v", s)
			}
			if s[1] == "" {
				return fmt.Errorf("parameter value can not be empty: %+v", s)
			}
			p[s[0]] = s[1]
		}
		i.Spec.Parameters = p
	}
	if _, err = kc.InstallInstanceObjToCluster(&i); err != nil {
		return errors.Wrapf(err, "installing %s-instance.yaml", name)
	}
	fmt.Printf("instance.%s/%s created\n", i.APIVersion, i.Name)
	return nil
}
