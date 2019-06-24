package repo

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/pkg/errors"
)

// FrameworkRepository represents a framework repository
type FrameworkRepository struct {
	Config *RepositoryConfiguration
	Client HTTPClient
}

// NewFrameworkRepository constructs FrameworkRepository
func NewFrameworkRepository(conf *RepositoryConfiguration) (*FrameworkRepository, error) {
	_, err := url.Parse(conf.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %s", conf.URL)
	}

	client, err := NewHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("could not construct http client: %v", err)
	}

	return &FrameworkRepository{
		Config: conf,
		Client: *client,
	}, nil
}

// DownloadIndexFile fetches the index file from a repository.
func (r *FrameworkRepository) DownloadIndexFile() (*IndexFile, error) {
	var indexURL string
	parsedURL, err := url.Parse(r.Config.URL)
	if err != nil {
		return nil, errors.Wrap(err, "parsing config url")
	}
	parsedURL.Path = fmt.Sprintf("%s/index.yaml", strings.TrimSuffix(parsedURL.Path, "/"))

	indexURL = parsedURL.String()

	resp, err := r.Client.Get(indexURL)
	if err != nil {
		return nil, errors.Wrap(err, "getting index url")
	}

	indexBytes, err := ioutil.ReadAll(resp)
	if err != nil {
		return nil, errors.Wrap(err, "reading index response")
	}

	indexFile, err := parseIndexFile(indexBytes)
	return indexFile, err
}

// GetPackage downloads the tgz file from the given repo
func (r *FrameworkRepository) GetPackage(packageName string) (*PackageCRDs, error) {
	var fileURL string
	parsedURL, err := url.Parse(r.Config.URL)
	if err != nil {
		return nil, errors.Wrap(err, "parsing config url")
	}
	parsedURL.Path = fmt.Sprintf("%s/%s.tgz", parsedURL.Path, packageName)

	fileURL = parsedURL.String()

	resp, err := r.Client.Get(fileURL)
	if err != nil {
		return nil, errors.Wrap(err, "getting file url")
	}

	// first try old package format
	fvPackage, err := ReadTarGzPackage(resp)

	if err != nil {
		return nil, fmt.Errorf("unable to parse the package. Errors are: %+v", err)
	}

	return fvPackage, nil
}

// GetFrameworkVersionDependencies helper method returns a slice of strings that contains the names of all
// dependency Frameworks
func GetFrameworkVersionDependencies(fv *v1alpha1.FrameworkVersion) ([]string, error) {
	var dependencyFrameworks []string
	if fv.Spec.Dependencies != nil {
		for _, v := range fv.Spec.Dependencies {
			dependencyFrameworks = append(dependencyFrameworks, v.Name)
		}
	}
	return dependencyFrameworks, nil
}
