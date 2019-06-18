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

// DownloadBundle downloads the tgz file from the given repo
func (r *FrameworkRepository) DownloadBundle(bundleName string) (*InstallCRDs, error) {
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
	fvPackage, err := ConvertV0Package(resp)

	if err != nil {
		// try to parse package as V1 package
		fvPackageV1, errV1 := PackageFromTarball(resp)
		if errV1 != nil {
			return nil, fmt.Errorf("unable to parse the package as V0 or V1 package. Errors are: V0 %v, V1 %v", err, errV1)
		}
		return fvPackageV1, nil
	}

	return fvPackage, nil
}

// GetFrameworkVersionDependencies returns a slice of strings that contains the names of all dependency Frameworks
func (r *FrameworkRepository) GetFrameworkVersionDependencies(name string, fv *v1alpha1.FrameworkVersion) ([]string, error) {
	var dependencyFrameworks []string
	if fv.Spec.Dependencies != nil {
		for _, v := range fv.Spec.Dependencies {
			dependencyFrameworks = append(dependencyFrameworks, v.Name)
		}
	}
	return dependencyFrameworks, nil
}

