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

	err = installFramework(args)
	if err != nil {
		return errors.WithMessage(err, "could not install framework")
	}

	return nil
}

func installFramework(args []string) error {

	if len(args) < 1 {
		return fmt.Errorf("no argument provided")
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

	// Check if all CRDs are installed
	err = check.KudoCRDs(k2c)
	if err != nil {
		return errors.Wrap(err, "checking kudo crd types")
	}

	for _, name := range args {
		// Check if Framework exists
		if !k2c.FrameworkExists(name) {
			return fmt.Errorf("framework crd for %s does not exist", name)
		}
		// Check if AnyFrameworkVersion for Framework exists
		if !k2c.AnyFrameworkVersionExists(name) {
			// Todo: install latest frameworkVersion
			return fmt.Errorf("frameworkversion crd for %s does not exist", name)
		}

		content, err := gc.GetMostRecentContentDir(name)
		if err != nil {
			return errors.Wrap(err, "sorting most recent content dir")
		}

		mostRecentVersion, err := gc.GetMostRecentFrameworkVersion(name, *content.Path)
		if err != nil {
			return errors.Wrap(err, "getting most recent frameworkversion version")
		}

		// Check if FrameworkVersion is out of sync with official FrameworkVersion for this Framework
		if !k2c.FrameworkVersionOutdated(name, mostRecentVersion) {
			// This happens when the most recent official FrameworkVersion is not existing. E.g.
			// when a version has been installed that is not part of the official kudobuilder/frameworks repo.
			if !vars.AutoApprove {
				fmt.Printf("No official FrameworkVersion has been found for \"%s\". "+
					"Do you want to install the most recent? (Yes/no) ", name)
				if helpers.AskForConfirmation() {
					// Todo: Install FrameworkVersion
				}
			}
			fmt.Printf("No official FrameworkVersion has been found, installing most recent...")
			// Todo: Install FrameworkVersion
		}

		fmt.Println(content)

	}

	return nil
}
