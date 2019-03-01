package github

import (
	"context"
	"fmt"
	"github.com/google/go-github/github" // with go modules disabled
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/helpers"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/pkg/errors"
	"os"
	"sigs.k8s.io/yaml"
	"strings"
)

type GithubClient struct {
	client *github.Client
}

// NewGithubClient generates a new Github client and returns an error if it failed.
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

// GetMostRecentFrameworkContentDir returns the content of the most recent Framework
func (g *GithubClient) GetMostRecentFrameworkContentDir(framework string) (*github.RepositoryContent, error) {
	var directoryContents []*github.RepositoryContent
	directoryContents, err := g.GetStableFrameworkContentDir(framework)
	if err != nil {
		directoryContents, err = g.GetIncubatingFrameworkContentDir(framework)
		if err != nil {
			return nil, errors.Wrap(err, "getting framework content")
		}
	}
	directoryContentsSorted, err := helpers.SortDirectoryContent(directoryContents)
	if err != nil {
		return nil, errors.Wrap(err, "sorting framework content")
	}

	mostRecentContentDir := directoryContentsSorted[0]

	return mostRecentContentDir, nil
}

// GetSpecificFrameworkContentDir returns the content of a specific Framework. If no Framework was found there will
// an error returned.
func (g *GithubClient) GetSpecificFrameworkContentDir(framework string) (*github.RepositoryContent, error) {
	var directoryContents []*github.RepositoryContent
	directoryContents, err := g.GetStableFrameworkContentDir(framework)
	if err != nil {
		directoryContents, err = g.GetIncubatingFrameworkContentDir(framework)
		if err != nil {
			return nil, errors.Wrap(err, "getting framework content")
		}
	}

	for k, v := range directoryContents {
		if vars.RepoVersion == *v.Name {
			return directoryContents[k], nil
		}
	}
	return nil, fmt.Errorf("no matching repo version found")
}

// GetStableFrameworkContentDir returns the content of a stable Framework. It returns an error if no Framework was
// found.
func (g *GithubClient) GetStableFrameworkContentDir(framework string) ([]*github.RepositoryContent, error) {
	_, directoryContents, _, err := g.client.Repositories.GetContents(context.Background(), "kudobuilder",
		"frameworks", "repo/stable/"+framework+"/versions", &github.RepositoryContentGetOptions{})
	if err != nil {
		switch err.(type) {
		case *github.ErrorResponse:
			errM := err.(*github.ErrorResponse)
			if errM.Response.StatusCode == 404 {
				return nil, errors.Wrap(err, "stable framework not found")
			}
		default:
			return nil, errors.Wrap(err, "getting stable framework")
		}
	}
	return directoryContents, nil
}

// GetIncubatingFrameworkContentDir returns the content of an incubating Framework. It returns an error if no Framework
// was found.
func (g *GithubClient) GetIncubatingFrameworkContentDir(framework string) ([]*github.RepositoryContent, error) {
	_, directoryContents, _, err := g.client.Repositories.GetContents(context.Background(), "kudobuilder",
		"frameworks", "repo/incubating/"+framework+"/versions", &github.RepositoryContentGetOptions{})
	if err != nil {
		switch err.(type) {
		case *github.ErrorResponse:
			errM := err.(*github.ErrorResponse)
			if errM.Response.StatusCode == 404 {
				return nil, errors.Wrap(err, "incubating framework not found")
			}
		default:
			return nil, errors.Wrap(err, "getting incubating framework")
		}
	}
	return directoryContents, nil
}

// GetFrameworkVersion returns the version to a given Framework
func (g *GithubClient) GetFrameworkVersion(name, path string) (string, error) {
	filePath := path + "/" + name + "-frameworkversion.yaml"
	filecontent, _, _, err := g.client.Repositories.GetContents(context.Background(), "kudobuilder",
		"frameworks", filePath, &github.RepositoryContentGetOptions{})
	if err != nil {
		switch err.(type) {
		case *github.ErrorResponse:
			errM := err.(*github.ErrorResponse)
			if errM.Response.StatusCode == 404 {
				return "", errors.Wrap(err, "frameworkversion not found")
			}
		default:
			return "", errors.Wrap(err, "getting frameworkversion")
		}
	}
	var fv v1alpha1.FrameworkVersion
	fileContentStr, err := filecontent.GetContent()
	if err != nil {
		return "", errors.Wrap(err, "getting frameworkversion content")
	}
	err = yaml.Unmarshal([]byte(fileContentStr), &fv)
	if err != nil {
		return "", errors.Wrap(err, "unmarshalling frameworkversion content")
	}

	if fv.Spec.Version == "" {
		return "", fmt.Errorf("cannot be empty")
	}

	return fv.Spec.Version, nil
}

// GetFrameworkYaml returns a Framework object from a given repo in the official GitHub repo
func (g *GithubClient) GetFrameworkYaml(name, path string) (*v1alpha1.Framework, error) {
	filePath := path + "/" + name + "-framework.yaml"
	fileContent, _, _, err := g.client.Repositories.GetContents(context.Background(), "kudobuilder",
		"frameworks", filePath, &github.RepositoryContentGetOptions{})
	if err != nil {
		switch err.(type) {
		case *github.ErrorResponse:
			errM := err.(*github.ErrorResponse)
			if errM.Response.StatusCode == 404 {
				return nil, errors.Wrapf(err, "%s-framework.yaml not found", name)
			}
		default:
			return nil, errors.Wrapf(err, "getting %s-framework.yaml", name)
		}
	}
	var f v1alpha1.Framework
	fileContentStr, err := fileContent.GetContent()
	if err != nil {
		return nil, errors.Wrapf(err, "getting %s-framework.yaml content", name)
	}
	err = yaml.Unmarshal([]byte(fileContentStr), &f)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshalling %s-framework.yaml content", name)
	}
	return &f, nil
}

// GetFrameworkVersionYaml returns a FrameworkVersion object from a given repo in the official GitHub repo
func (g *GithubClient) GetFrameworkVersionYaml(name, path string) (*v1alpha1.FrameworkVersion, error) {
	filePath := path + "/" + name + "-frameworkversion.yaml"
	fileContent, _, _, err := g.client.Repositories.GetContents(context.Background(), "kudobuilder",
		"frameworks", filePath, &github.RepositoryContentGetOptions{})
	if err != nil {
		switch err.(type) {
		case *github.ErrorResponse:
			errM := err.(*github.ErrorResponse)
			if errM.Response.StatusCode == 404 {
				return nil, errors.Wrapf(err, "%s-frameworkversion.yaml not found", name)
			}
		default:
			return nil, errors.Wrapf(err, "getting %s-frameworkversion.yaml", name)
		}
	}
	var f v1alpha1.FrameworkVersion
	fileContentStr, err := fileContent.GetContent()
	if err != nil {
		return nil, errors.Wrapf(err, "getting %s-frameworkversion.yaml content", name)
	}
	err = yaml.Unmarshal([]byte(fileContentStr), &f)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshalling %s-frameworkversion.yaml content", name)
	}
	return &f, nil
}

// GetInstanceYaml returns a Instance object from a given repo in the official GitHub repo
func (g *GithubClient) GetInstanceYaml(name, path string) (*v1alpha1.Instance, error) {
	filePath := path + "/" + name + "-instance.yaml"
	fileContent, _, _, err := g.client.Repositories.GetContents(context.Background(), "kudobuilder",
		"frameworks", filePath, &github.RepositoryContentGetOptions{})
	if err != nil {
		switch err.(type) {
		case *github.ErrorResponse:
			errM := err.(*github.ErrorResponse)
			if errM.Response.StatusCode == 404 {
				return nil, errors.Wrapf(err, "%s-instance.yaml not found", name)
			}
		default:
			return nil, errors.Wrapf(err, "getting %s-instance.yaml", name)
		}
	}
	var f v1alpha1.Instance
	fileContentStr, err := fileContent.GetContent()
	if err != nil {
		return nil, errors.Wrapf(err, "getting %s-instance.yaml content", name)
	}
	err = yaml.Unmarshal([]byte(fileContentStr), &f)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshalling %s-instance.yaml content", name)
	}
	return &f, nil
}

// FrameworkVersionDependencyExists returns a slice of strings that contains the names of all dependency Frameworks
// from a given repo in the official GitHub repo
func (g *GithubClient) GetFrameworkVersionDependencies(name, path string) ([]string, error) {
	filePath := path + "/" + name + "-frameworkversion.yaml"
	fileContent, _, _, err := g.client.Repositories.GetContents(context.Background(), "kudobuilder",
		"frameworks", filePath, &github.RepositoryContentGetOptions{})
	if err != nil {
		switch err.(type) {
		case *github.ErrorResponse:
			errM := err.(*github.ErrorResponse)
			if errM.Response.StatusCode == 404 {
				return nil, errors.Wrapf(err, "%s-frameworkversion.yaml not found", name)
			}
		default:
			return nil, errors.Wrapf(err, "getting %s-frameworkversion.yaml", name)
		}
	}
	var f v1alpha1.FrameworkVersion
	fileContentStr, err := fileContent.GetContent()
	if err != nil {
		return nil, errors.Wrapf(err, "getting %s-frameworkversion.yaml content", name)
	}
	err = yaml.Unmarshal([]byte(fileContentStr), &f)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshalling %s-frameworkversion.yaml content", name)
	}
	var dependencyFrameworks []string
	if f.Spec.Dependencies != nil {
		for _, v := range f.Spec.Dependencies {
			dependencyFrameworks = append(dependencyFrameworks, v.Name)
		}
	}
	return dependencyFrameworks, nil
}
