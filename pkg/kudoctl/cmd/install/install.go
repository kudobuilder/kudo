package install

import (
	"fmt"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/check"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/github"
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

	kc, err := k8s.NewK8sClient()
	if err != nil {
		return errors.Wrap(err, "creating kubernetes client")
	}

	for _, name := range args {
		err = kc.FrameworkExists(name)
		if err != nil {
			return errors.Wrap(err, "checking framework")
		}

		content, err := gc.GetMostRecentContentDir(name)
		if err != nil {
			return errors.Wrap(err, "sorting most recent content dir")
		}

		fmt.Println(content)

	}

	return nil
}
