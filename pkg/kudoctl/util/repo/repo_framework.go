package repo

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

// FrameworkRepository represents a framework repository
type FrameworkRepository struct {
	Config         *RepositoryConfiguration
	FrameworkPaths []string
	Client         HTTPClient
}

// BufferedFile represents an archive file buffered for later processing.
type BufferedFile struct {
	Name string
	Data []byte
}

// NewFrameworkRepository constructs FrameworkRepository
func NewFrameworkRepository(cfg *RepositoryConfiguration) (*FrameworkRepository, error) {
	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid chart URL format: %s", cfg.URL)
	}

	client, err := NewHTTPClient(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("could not construct protocol handler for: %s error: %v", u.Scheme, err)
	}

	return &FrameworkRepository{
		Config: cfg,
		Client: *client,
	}, nil
}

// DownloadIndexFile fetches the index from a repository.
//
// cachePath is prepended to any index that does not have an absolute path. This
// is for pre-2.2.0 repo files.
func (r *FrameworkRepository) DownloadIndexFile() (*IndexFile, error) {
	var indexURL string
	parsedURL, err := url.Parse(r.Config.URL)
	if err != nil {
		return nil, errors.Wrap(err, "parsing config url")
	}
	parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/") + "/index.yaml"

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

// DownloadBundleFile downloads the tgz file from the given repo
func (r *FrameworkRepository) DownloadBundleFile(bundle string) error {
	var fileURL string
	parsedURL, err := url.Parse(r.Config.URL)
	if err != nil {
		return errors.Wrap(err, "parsing config url")
	}
	parsedURL.Path = parsedURL.Path + "/" + bundle + ".tgz"

	fileURL = parsedURL.String()

	resp, err := r.Client.Get(fileURL)
	if err != nil {
		return errors.Wrap(err, "getting file url")
	}

	err = untar(vars.RepoPath+"/"+bundle, resp)
	if err != nil {
		return errors.Wrapf(err, "failed unpacking %s", bundle)
	}

	return nil
}

// GetFrameworkVersion gets the proper Framework version of a given Framework
func (r *FrameworkRepository) GetFrameworkVersion(name, path string) (string, error) {
	frameworkVersionPath := path + "/" + name + "-frameworkversion.yaml"
	frameworkVersionYamlFile, err := os.Open(frameworkVersionPath)
	if err != nil {
		return "", errors.Wrap(err, "failed opening frameworkversion file")
	}

	frameworkVersionByteValue, err := ioutil.ReadAll(frameworkVersionYamlFile)
	if err != nil {
		return "", errors.Wrap(err, "failed reading frameworkversion file")
	}

	var fv v1alpha1.FrameworkVersion
	err = yaml.Unmarshal(frameworkVersionByteValue, &fv)
	if err != nil {
		return "", errors.Wrapf(err, "unmarshalling %s-frameworkversion.yaml content", name)
	}
	return fv.Spec.Version, nil
}

// GetFrameworkVersionDependencies returns a slice of strings that contains the names of all dependency Frameworks
// from a given repo in the official GitHub repo
func (r *FrameworkRepository) GetFrameworkVersionDependencies(name, path string) ([]string, error) {
	frameworkVersionPath := path + "/" + name + "-frameworkversion.yaml"
	frameworkVersionYamlFile, err := os.Open(frameworkVersionPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed opening frameworkversion file")
	}

	frameworkVersionByteValue, err := ioutil.ReadAll(frameworkVersionYamlFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed reading frameworkversion file")
	}

	var fv v1alpha1.FrameworkVersion
	err = yaml.Unmarshal(frameworkVersionByteValue, &fv)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshalling %s-frameworkversion.yaml content", name)
	}
	var dependencyFrameworks []string
	if fv.Spec.Dependencies != nil {
		for _, v := range fv.Spec.Dependencies {
			dependencyFrameworks = append(dependencyFrameworks, v.Name)
		}
	}
	return dependencyFrameworks, nil
}

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
// example creation: tar -zcvf kafka-0.1.0.tgz *
func untar(dst string, r io.Reader) error {

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:

			err := os.MkdirAll(filepath.Dir(target), 0755)
			if err != nil {
				return errors.Wrapf(err, "making directory for file %v", target)
			}

			out, err := os.Create(target)
			if err != nil {
				return errors.Wrapf(err, "creating new file %v", target)
			}
			defer out.Close()

			err = out.Chmod(os.FileMode(header.Mode))
			if err != nil && runtime.GOOS != "windows" {
				return errors.Wrapf(err, "changing file %v", target)
			}

			_, err = io.Copy(out, tr)
			if err != nil {
				return errors.Wrapf(err, "writing file %v", target)
			}
		}
	}
}
