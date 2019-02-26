package install

import (
	"context"
	"fmt"
	gg "github.com/google/go-github/github" // with go modules disabled
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/check"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/github"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"sort"
	"strconv"
)

func InstallCmd(cmd *cobra.Command, args []string) error {

	// Validating flags
	// this makes --frameworkname mandatory
	frameworkNameFlag, err := cmd.Flags().GetString("frameworkname")
	if err != nil || frameworkNameFlag == "" {
		return fmt.Errorf("Please set Frameworkname flag, e.g. \"--frameworkname=kafka\"")
	}

	_, err = cmd.Flags().GetString("kubeconfig")
	// This makes --kubeconfig flag optional
	if err != nil {
		return fmt.Errorf("Please set --kubeconfig flag")
	}

	err = check.KubeConfigPath()
	if err != nil {
		return errors.WithMessage(err, "could not check kubeconfig path")
	}

	err = install(args)
	if err != nil {
		return errors.WithMessage(err, "could not install")
	}

	return nil
}

func install(args []string) error {

	err := check.GithubCredentials()
	if err != nil {
		return errors.WithMessage(err, "could not check github credential path")
	}

	cred, err := github.GetGithubCredentials()
	if err != nil {
		return errors.WithMessage(err, "could not get github credential")
	}

	_, err = clientcmd.BuildConfigFromFlags("", vars.KubeConfigPath)
	if err != nil {
		return errors.Wrap(err, "getting config failed")
	}

	gc, err := github.NewGithubClient(cred)
	if err != nil {
		return errors.Wrap(err, "creating github client")
	}

	_, directoryContents, _, err := gc.Repositories.GetContents(context.Background(), "kudobuilder", "frameworks", "repo/stable/"+vars.FrameworkName+"/versions", &gg.RepositoryContentGetOptions{})
	if err != nil {
		switch err.(type) {
		case *gg.ErrorResponse:
			errM := err.(*gg.ErrorResponse)
			if errM.Response.StatusCode == 404 {
				// Todo: try incubating repos
				return errors.Wrap(err, "github repo not found")
			}
		default:
			return errors.Wrap(err, "getting github content")
		}
	}

	// Sorting with highest number first
	sort.Slice(directoryContents, func(i, j int) bool {
		v1, _ := strconv.Atoi(*directoryContents[i].Name)
		v2, _ := strconv.Atoi(*directoryContents[j].Name)
		return v1 > v2
	})

	return nil
}
