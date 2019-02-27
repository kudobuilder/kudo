package github

import (
	"context"
	"fmt"
	"github.com/google/go-github/github" // with go modules disabled
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/helpers"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"os"
	"strings"
)

type GithubClient struct {
	client *github.Client
}

func NewGithubClient(cred string) (*GithubClient, error) {

	// Giving the option to set a Github user and password
	gitUser := os.Getenv("GIT_USER")
	gitPassword := os.Getenv("GIT_PASSWORD")

	result := strings.Split(cred, ":")

	if gitUser == "" {
		gitUser = result[0]
	}

	if gitPassword == "" {
		gitPassword = result[1]
	}

	tp := github.BasicAuthTransport{
		Username: gitUser,
		Password: gitPassword,
	}

	client := github.NewClient(tp.Client())
	ctx := context.Background()
	_, _, err := client.Users.Get(ctx, "")

	// Is this a two-factor auth error? If so, set cred as OTP token
	if _, ok := err.(*github.TwoFactorAuthError); ok {
		tp.OTP = cred
	}

	return &GithubClient{client: client}, nil
}

func (g *GithubClient) GetMostRecentContentDir(framework string) (*github.RepositoryContent, error) {
	_, directoryContents, _, err := g.client.Repositories.GetContents(context.Background(), "kudobuilder", "frameworks", "repo/stable/"+framework+"/versions", &github.RepositoryContentGetOptions{})
	if err != nil {
		switch err.(type) {
		case *github.ErrorResponse:
			errM := err.(*github.ErrorResponse)
			if errM.Response.StatusCode == 404 {
				// Todo: try incubating repos
				return nil, errors.Wrap(err, "github repo not found")
			}
		default:
			return nil, errors.Wrap(err, "getting github content")
		}
	}
	directoryContentsSorted, err := helpers.SortDirectoryContent(directoryContents)
	if err != nil {
		return nil, errors.Wrap(err, "sorting dir content")
	}

	mostRecentContentDir := directoryContentsSorted[0]

	return mostRecentContentDir, nil
}

func (g *GithubClient) GetMostRecentFrameworkVersion(name, path string) (string, error) {
	filePath := path + "/" + name + "-frameworkversion.yaml"
	filecontent, _, _, err := g.client.Repositories.GetContents(context.Background(), "kudobuilder", "frameworks", filePath, &github.RepositoryContentGetOptions{})
	if err != nil {
		switch err.(type) {
		case *github.ErrorResponse:
			errM := err.(*github.ErrorResponse)
			if errM.Response.StatusCode == 404 {
				return "", errors.Wrap(err, "github repo not found")
			}
		default:
			return "", errors.Wrap(err, "getting github content")
		}
	}
	var fv v1alpha1.FrameworkVersion
	fileContentStr, err := filecontent.GetContent()
	if err != nil {
		return "", errors.Wrap(err, "getting file content")
	}
	err = yaml.Unmarshal([]byte(fileContentStr), &fv)
	if err != nil {
		return "", errors.Wrap(err, "unmarshalling file content")
	}

	if fv.Spec.Version == "" {
		return "", fmt.Errorf("cannot be empty")
	}

	return fv.Spec.Version, nil
}
