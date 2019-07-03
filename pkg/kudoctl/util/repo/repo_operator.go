package repo

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/bundle"
	"github.com/kudobuilder/kudo/pkg/kudoctl/http"
	"github.com/pkg/errors"
)

// Repository is a abstraction for a service that can retrieve package bundles
type Repository interface {
	GetPackageReader(name string, version string) (io.Reader, error)
	GetPackageBundle(name string, version string) (bundle.Bundle, error)
}

// OperatorRepository represents a operator repository
type OperatorRepository struct {
	Config *RepositoryConfiguration
	Client http.Client
}

// NewOperatorRepository constructs OperatorRepository
func NewOperatorRepository(conf *RepositoryConfiguration) (*OperatorRepository, error) {
	_, err := url.Parse(conf.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %s", conf.URL)
	}

	client, err := http.NewClient()
	if err != nil {
		return nil, fmt.Errorf("could not construct http client: %v", err)
	}

	return &OperatorRepository{
		Config: conf,
		Client: *client,
	}, nil
}

// downloadIndexFile fetches the index file from a repository.
func (r *OperatorRepository) downloadIndexFile() (*IndexFile, error) {
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

// getPackageReaderByFullPackageName downloads the tgz file from the remote repository and unmarshals it to the package CRDs
func (r *OperatorRepository) getPackageReaderByFullPackageName(fullPackageName string) (io.Reader, error) {
	var fileURL string
	parsedURL, err := url.Parse(r.Config.URL)
	if err != nil {
		return nil, errors.Wrap(err, "parsing config url")
	}
	parsedURL.Path = fmt.Sprintf("%s/%s.tgz", parsedURL.Path, fullPackageName)

	fileURL = parsedURL.String()
	return r.getPackageReaderByURL(fileURL)
}

func (r *OperatorRepository) getPackageReaderByURL(packageURL string) (io.Reader, error) {
	resp, err := r.Client.Get(packageURL)
	if err != nil {
		return nil, errors.Wrap(err, "getting package url")
	}

	return resp, nil
}

// GetPackageReader provides an io.Reader for a provided package name and optional version
func (r *OperatorRepository) GetPackageReader(name string, version string) (io.Reader, error) {

	// Construct the package name and download the index file from the remote repo
	indexFile, err := r.downloadIndexFile()
	if err != nil {
		return nil, errors.WithMessage(err, "could not download repository index file")
	}

	var bundleVersion *BundleVersion

	if version == "" {
		bv, err := indexFile.GetByName(name)
		if err != nil {
			return nil, errors.Wrapf(err, "getting %s in index file", name)
		}
		bundleVersion = bv
	} else {
		bv, err := indexFile.GetByNameAndVersion(name, version)
		if err != nil {
			return nil, errors.Wrapf(err, "getting %s in index file", name)
		}
		bundleVersion = bv
	}

	packageName := bundleVersion.Name + "-" + bundleVersion.Version

	return r.getPackageReaderByFullPackageName(packageName)
}

// GetPackageBundle provides an Bundle for a provided package name and optional version
func (r *OperatorRepository) GetPackageBundle(name string, version string) (bundle.Bundle, error) {
	// check to see if name is url
	if isValidURL(name) {
		reader, err := r.getPackageReaderByURL(name)
		if err != nil {
			return nil, err
		}
		return bundle.NewBundleFromReader(reader), nil
	}
	reader, err := r.GetPackageReader(name, version)
	if err != nil {
		return nil, err
	}
	return bundle.NewBundleFromReader(reader), nil
}

func isValidURL(name string) bool {
	_, err := url.ParseRequestURI(name)
	return err == nil
}

// GetOperatorVersionDependencies helper method returns a slice of strings that contains the names of all
// dependency Operators
func GetOperatorVersionDependencies(fv *v1alpha1.OperatorVersion) ([]string, error) {
	var dependencyOperators []string
	if fv.Spec.Dependencies != nil {
		for _, v := range fv.Spec.Dependencies {
			dependencyOperators = append(dependencyOperators, v.Name)
		}
	}
	return dependencyOperators, nil
}
