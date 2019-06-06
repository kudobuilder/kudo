package repo

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
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

const (
	frameworkFileName = "-framework.yaml"
	versionFileName   = "-frameworkversion.yaml"
	instanceFileName  = "-instance.yaml"
)

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
func (r *FrameworkRepository) DownloadBundle(bundleName string) (*FrameworkBundle, error) {
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

	bundle, err := untar(resp)

	if err != nil {
		return nil, errors.Wrapf(err, "failed unpacking %s", bundleName)
	}

	return bundle, nil
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

func untar(r io.Reader) (*FrameworkBundle, error) {

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := gzr.Close()
		if err != nil {
			fmt.Printf("Error when closing gzip reader %s", err)
		}
	}()

	tr := tar.NewReader(gzr)

	result := &FrameworkBundle{}
	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			if result.Instance != nil && result.FrameworkVersion != nil && result.Framework != nil {
				// bundle is complete
				return result, nil
			}

			var missing []string
			if result.Instance == nil {
				missing = append(missing, "instance.yaml")
			} else if result.FrameworkVersion != nil {
				missing = append(missing, "frameworkversion.yaml")
			} else if result.Framework != nil {
				missing = append(missing, "framework.yaml")
			}
			return nil, fmt.Errorf("incomplete bundle - these files are missing: %v", missing)

		// return any other error
		case err != nil:
			return nil, err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// check the file type
		switch header.Typeflag {

		case tar.TypeDir:
			// we don't handle folders right now, the structure is flat

		// if it's a file create it
		case tar.TypeReg:
			bytes, err := ioutil.ReadAll(tr)
			if err != nil {
				return nil, errors.Wrapf(err, "while reading file from bundle tarball %s", header.Name)
			}

			switch {
			case isFrameworkFile(header.Name):
				var f v1alpha1.Framework
				if err = yaml.Unmarshal(bytes, &f); err != nil {
					return nil, errors.Wrapf(err, "unmarshalling %s content", header.Name)
				}
				result.Framework = &f
			case isVersionFile(header.Name):
				var fv v1alpha1.FrameworkVersion
				if err = yaml.Unmarshal(bytes, &fv); err != nil {
					return nil, errors.Wrapf(err, "unmarshalling %s content", header.Name)
				}
				result.FrameworkVersion = &fv
			case isInstanceFile(header.Name):
				var i v1alpha1.Instance
				if err = yaml.Unmarshal(bytes, &i); err != nil {
					return nil, errors.Wrapf(err, "unmarshalling %s content", header.Name)
				}
				result.Instance = &i
			default:
				return nil, fmt.Errorf("unexpected file in the tarball structure %s", header.Name)
			}
		}
	}
}

func isFrameworkFile(name string) bool {
	return strings.HasSuffix(name, frameworkFileName)
}

func isVersionFile(name string) bool {
	return strings.HasSuffix(name, versionFileName)
}

func isInstanceFile(name string) bool {
	return strings.HasSuffix(name, instanceFileName)
}
