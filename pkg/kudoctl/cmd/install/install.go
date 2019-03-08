package install

import (
	"fmt"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/check"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/github"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/helpers"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
)

func InstallCmd(cmd *cobra.Command, args []string) error {

	_, err := cmd.Flags().GetString("kubeconfig")
	// This makes --kubeconfig flag optional
	if err != nil {
		return fmt.Errorf("get flag: %+v", err)
	}

	err = check.KubeConfigPath()
	if err != nil {
		return errors.WithMessage(err, "could not check kubeconfig path")
	}

	err = installFrameworks(args)
	if err != nil {
		return errors.WithMessage(err, "could not install framework(s)")
	}

	return nil
}

func installFrameworks(args []string) error {

	if len(args) < 1 {
		return fmt.Errorf("no argument provided")
	}

	if len(args) > 1 && vars.PackageVersion != "" {
		return fmt.Errorf("--repo-version not supported in multi framework install")
	}

	err := check.GithubCredentials()
	if err != nil {
		return errors.WithMessage(err, "could not check github credential path")
	}

	cred, err := github.GetGithubCredentials()
	if err != nil {
		return errors.WithMessage(err, "could not get github credential")
	}

	gc, err := github.NewGithubClient(cred)
	if err != nil {
		return errors.Wrap(err, "creating github client")
	}

	_, err = clientcmd.BuildConfigFromFlags("", vars.KubeConfigPath)
	if err != nil {
		return errors.Wrap(err, "getting config failed")
	}

	kudoclient, err := kudo.NewKudoClient()
	if err != nil {
		return errors.Wrap(err, "creating kudo client")
	}

	for _, name := range args {
		err := installSingleFramework(name, "", gc, kudoclient)
		if err != nil {
			return err
		}
	}
	return nil
}

// Todo: needs testing
// installSingleFramework is the umbrella for a single framework installation that gathers the business logic
// for a cluster and returns an error in case there is a problem
func installSingleFramework(name, previous string, gc *github.GithubClient, k2c *kudo.KudoClient) error {
	// Get most recent ContentDir for selected Framework
	content, err := gc.GetMostRecentFrameworkContentDir(name)
	if err != nil {
		return errors.Wrap(err, "sorting most recent content dir")
	}

	if vars.PackageVersion != "" {
		content, err = gc.GetSpecificFrameworkContentDir(name)
		if err != nil {
			return errors.Wrap(err, "getting specific content dir")
		}
	}

	// Framework part

	// Check if Framework exists
	if !k2c.FrameworkExistsInCluster(name) {
		err := installSingleFrameworkToCluster(name, *content.Path, gc, k2c)
		if err != nil {
			return errors.Wrap(err, "installing single Framework")
		}
	}

	// FrameworkVersion part

	// Get the string of the version in FrameworkVersion of a selected Framework
	frameworkVersion, err := gc.GetFrameworkVersion(name, *content.Path)
	if err != nil {
		return errors.Wrap(err, "getting most recent FrameworkVersion version")
	}

	// Check if AnyFrameworkVersion for Framework exists
	if !k2c.AnyFrameworkVersionExistsInCluster(name) {
		// FrameworkVersion CRD for Framework does not exist
		err := installSingleFrameworkVersionToCluster(name, *content.Path, gc, k2c)
		if err != nil {
			return errors.Wrap(err, "installing single FrameworkVersion")
		}
	}

	// Check if FrameworkVersion is out of sync with official FrameworkVersion for this Framework
	if !k2c.FrameworkVersionInClusterOutOfSync(name, frameworkVersion) {
		// This happens when the given FrameworkVersion is not existing. E.g.
		// when a version has been installed that is not part of the official kudobuilder/frameworks repo.
		if !vars.AutoApprove {
			fmt.Printf("No official FrameworkVersion has been found for \"%s\". "+
				"Do you want to install one? (Yes/no) ", name)
			if helpers.AskForConfirmation() {
				err := installSingleFrameworkVersionToCluster(name, *content.Path, gc, k2c)
				if err != nil {
					return errors.Wrap(err, "installing single FrameworkVersion")
				}
			}
		} else {
			err := installSingleFrameworkVersionToCluster(name, *content.Path, gc, k2c)
			if err != nil {
				return errors.Wrap(err, "installing single FrameworkVersion")
			}
		}

	}

	// Dependencies of the particular FrameworkVersion
	if vars.AllDependencies {
		dependencyFrameworks, err := gc.GetFrameworkVersionDependencies(name, *content.Path)
		if err != nil {
			return errors.Wrap(err, "getting Framework dependencies")
		}
		for _, v := range dependencyFrameworks {
			// recursive function call
			// Dependencies should not be as big as that they will have an overflow in the function stack frame
			// installSingleFramework makes sure that dependency Frameworks are created before the Framework itself
			// and it allows to inherit dependencies.
			err := installSingleFramework(v, name, gc, k2c)
			if err != nil {
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
	if !k2c.AnyInstanceExistsInCluster(name, frameworkVersion) {
		// This happens when the given FrameworkVersion is not existing. E.g.
		// when a version has been installed that is not part of the official kudobuilder/frameworks repo.
		if !vars.AutoApprove {
			fmt.Printf("No Instance tied to this \"%s\" version has been found. "+
				"Do you want to create one? (Yes/no) ", name)
			if helpers.AskForConfirmation() {
				// If Instance is a dependency we need to make sure installSingleInstanceToCluster is aware of it.
				// By having the previous string set we can make this distinction.
				err := installSingleInstanceToCluster(name, previous, *content.Path, gc, k2c)
				if err != nil {
					return errors.Wrap(err, "installing single Instance")
				}
			}
		} else {
			err := installSingleInstanceToCluster(name, previous, *content.Path, gc, k2c)
			if err != nil {
				return errors.Wrap(err, "installing single Instance")
			}
		}

	}

	return nil
}

// Todo: needs testing
// installSingleFrameworkToCluster installs a given Framework to the cluster
func installSingleFrameworkToCluster(name, path string, gc *github.GithubClient, k2c *kudo.KudoClient) error {
	frameworkYaml, err := gc.GetFrameworkYaml(name, path)
	if err != nil {
		return errors.Wrapf(err, "getting %s-framework.yaml", name)
	}
	_, err = k2c.InstallFrameworkObjToCluster(frameworkYaml)
	if err != nil {
		return errors.Wrapf(err, "installing %s-framework.yaml", name)
	}
	fmt.Printf("framework.%s/%s created\n", frameworkYaml.APIVersion, frameworkYaml.Name)
	return nil
}

// Todo: needs testing
// installSingleFrameworkVersionToCluster installs a given FrameworkVersion to the cluster
func installSingleFrameworkVersionToCluster(name, path string, gc *github.GithubClient, k2c *kudo.KudoClient) error {
	frameworkVersionYaml, err := gc.GetFrameworkVersionYaml(name, path)
	if err != nil {
		return errors.Wrapf(err, "getting %s-framework.yaml", name)
	}
	_, err = k2c.InstallFrameworkVersionObjToCluster(frameworkVersionYaml)
	if err != nil {
		return errors.Wrapf(err, "installing %s-framework.yaml", name)
	}
	fmt.Printf("frameworkversion.%s/%s created\n", frameworkVersionYaml.APIVersion, frameworkVersionYaml.Name)
	return nil
}

// Todo: needs testing
// installSingleInstanceToCluster installs a given Instance to the cluster
func installSingleInstanceToCluster(name, previous, path string, gc *github.GithubClient, k2c *kudo.KudoClient) error {
	instanceYaml, err := gc.GetInstanceYaml(name, path)
	if err != nil {
		return errors.Wrapf(err, "getting %s-instance.yaml", name)
	}
	// Customizing Instance
	// TODO: traversing, e.g. check function that looksup if key exists in the current FrameworkVersion
	// That way just Parameters will be applied if they exist in the matching FrameworkVersion
	// More checking required
	// E.g. when installing with flag --all-dependencies to prevent overwriting dependency Instance name

	// This checks if flag --instance was set with a name and it is the not a dependency Instance
	if vars.Instance != "" && previous == "" {
		instanceYaml.ObjectMeta.SetName(vars.Instance)
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
		instanceYaml.Spec.Parameters = p
	}
	_, err = k2c.InstallInstanceObjToCluster(instanceYaml)
	if err != nil {
		return errors.Wrapf(err, "installing %s-instance.yaml", name)
	}
	fmt.Printf("instance.%s/%s created\n", instanceYaml.APIVersion, instanceYaml.Name)
	return nil
}
