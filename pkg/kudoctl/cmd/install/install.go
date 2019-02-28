package install

import (
	"fmt"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/check"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/github"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/helpers"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/k8s"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

func InstallCmd(cmd *cobra.Command, args []string) error {

	// Validating flags
	/*
		// this makes --frameworkname mandatory
		frameworkNameFlag, err := cmd.Flags().GetString("frameworkname")
		if err != nil || frameworkNameFlag == "" {
			return fmt.Errorf("Please set Frameworkname flag, e.g. \"--frameworkname=kafka\"")
		}
	*/
	_, err := cmd.Flags().GetString("kubeconfig")
	// This makes --kubeconfig flag optional
	if err != nil {
		return fmt.Errorf("Please set --kubeconfig flag")
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

	if len(args) > 1 && vars.RepoVersion != "" {
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

	k2c, err := k8s.NewK2oClient()
	if err != nil {
		return errors.Wrap(err, "creating kudo client")
	}

	// SanityCheck if all CRDs are installed
	err = check.KudoCRDs(k2c)
	if err != nil {
		return errors.Wrap(err, "checking kudo crd types")
	}

	for _, name := range args {
		err := installSingleFramework(name, gc, k2c)
		if err != nil {
			return err
		}
	}
	return nil
}

func installSingleFramework(name string, gc *github.GithubClient, k2c *k8s.K2oClient) error {
	// Get most recent ContentDir for selected Framework
	content, err := gc.GetMostRecentFrameworkContentDir(name)
	if err != nil {
		return errors.Wrap(err, "sorting most recent content dir")
	}

	if vars.RepoVersion != "" {
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
		// This happens when the most recent official FrameworkVersion is not existing. E.g.
		// when a version has been installed that is not part of the official kudobuilder/frameworks repo.
		if !vars.AutoApprove {
			fmt.Printf("No official FrameworkVersion has been found for \"%s\". "+
				"Do you want to install the most recent? (Yes/no) ", name)
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

	if vars.AllDependencies {
		dependencyFrameworks, err := gc.GetFrameworkVersionDependencies(name, *content.Path)
		if err != nil {
			return errors.Wrap(err, "getting Framework dependencies")
		}
		for _, v := range dependencyFrameworks {
			err := installSingleFramework(v, gc, k2c)
			if err != nil {
				return errors.Wrapf(err, "installing dependency Framework %s", v)
			}
		}
	}
	return nil
}

func installSingleFrameworkToCluster(name, path string, gc *github.GithubClient, k2c *k8s.K2oClient) error {
	frameworkYaml, err := gc.GetFrameworkYaml(name, path)
	if err != nil {
		return errors.Wrapf(err, "getting %s-framework.yaml", name)
	}
	_, err = k2c.InstallFrameworkYamlToCluster(frameworkYaml)
	if err != nil {
		return errors.Wrapf(err, "installing %s-framework.yaml", name)
	}
	fmt.Printf("framework.%s/%s created\n", frameworkYaml.APIVersion, frameworkYaml.Name)
	return nil
}

func installSingleFrameworkVersionToCluster(name, path string, gc *github.GithubClient, k2c *k8s.K2oClient) error {
	frameworkVersionYaml, err := gc.GetFrameworkVersionYaml(name, path)
	if err != nil {
		return errors.Wrapf(err, "getting %s-framework.yaml", name)
	}
	_, err = k2c.InstallFrameworkVersionYamlToCluster(frameworkVersionYaml)
	if err != nil {
		return errors.Wrapf(err, "installing %s-framework.yaml", name)
	}
	fmt.Printf("frameworkversion.%s/%s created\n", frameworkVersionYaml.APIVersion, frameworkVersionYaml.Name)
	return nil
}
