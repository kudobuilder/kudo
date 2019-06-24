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
	Config    *RepositoryConfiguration
	Client    HTTPClient
	IndexFile *IndexFile
}

// FrameworkBundle contains parsed files from the framework bundle
type FrameworkBundle struct {
	Framework        *v1alpha1.Framework
	FrameworkVersion *v1alpha1.FrameworkVersion
	Instance         *v1alpha1.Instance
}

// NewFrameworkRepository constructs FrameworkRepository
func NewFrameworkRepository(cfg *RepositoryConfiguration) (*FrameworkRepository, error) {
	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid chart URL format: %s", cfg.URL)
	}

	client, err := NewHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("could not construct protocol handler for: %s error: %v", u.Scheme, err)
	}

	return &FrameworkRepository{
		Config: cfg,
		Client: *client,
	}, nil
}

// DownloadIndexFile fetches the index file from a repository. It has to be called manually to be able to defer its
// initialisation  based on whether framework installation is done from local tar.gz/folder or from a remote repo.
func (r *FrameworkRepository) DownloadIndexFile() error {
	var indexURL string
	parsedURL, err := url.Parse(r.Config.URL)
	if err != nil {
		return errors.Wrap(err, "parsing config url")
	}
	parsedURL.Path = fmt.Sprintf("%s/index.yaml", strings.TrimSuffix(parsedURL.Path, "/"))

	indexURL = parsedURL.String()

	resp, err := r.Client.Get(indexURL)
	if err != nil {
		return errors.Wrap(err, "getting index url")
	}

	indexBytes, err := ioutil.ReadAll(resp)
	if err != nil {
		return errors.Wrap(err, "reading index response")
	}

	r.IndexFile, err = parseIndexFile(indexBytes)
	return err
}

// GetPackage downloads the tgz file from the given repo
func (r *FrameworkRepository) GetPackage(bundleName string) (*PackageCRDs, error) {
	var fileURL string
	parsedURL, err := url.Parse(r.Config.URL)
	if err != nil {
		return nil, errors.Wrap(err, "parsing config url")
	}
	parsedURL.Path = fmt.Sprintf("%s/%s.tgz", parsedURL.Path, bundleName)

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
