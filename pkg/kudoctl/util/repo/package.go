package repo

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"sigs.k8s.io/yaml"
	"strings"
)

const (
	frameworkV0FileName = "-framework.yaml"
	versionV0FileName   = "-frameworkversion.yaml"
	instanceV0FileName  = "-instance.yaml"

	frameworkV1FileName = "framework.yaml"
	templateV1FileName = "templates/"
)

type FrameworkPackage interface {
	GetFrameworkCRD() (*v1alpha1.Framework, error)
	GetFrameworkVersionCRD() (*v1alpha1.FrameworkVersion, error)
	GetInstanceCRD() (*v1alpha1.Instance, error)
	ValidationError() error
}

type V0Package struct {
	Framework        *v1alpha1.Framework
	FrameworkVersion *v1alpha1.FrameworkVersion
	Instance         *v1alpha1.Instance
}

func (p *V0Package) GetFrameworkCRD() (*v1alpha1.Framework, error) { return p.Framework, nil }
func (p *V0Package) GetFrameworkVersionCRD() (*v1alpha1.FrameworkVersion, error) { return p.FrameworkVersion, nil }
func (p *V0Package) GetInstanceCRD() (*v1alpha1.Instance, error) { return p.Instance, nil }
func (p *V0Package) ValidationError() error {
	if p.Instance != nil && p.FrameworkVersion != nil && p.Framework != nil {
		return nil
	}
	var missing []string
	if p.Instance == nil {
		missing = append(missing, "instance.yaml")
	} else if p.FrameworkVersion != nil {
		missing = append(missing, "frameworkversion.yaml")
	} else if p.Framework != nil {
		missing = append(missing, "framework.yaml")
	}
	return fmt.Errorf("incomplete package - these files are missing: %v", missing)
}

func UntarV0Package(r io.Reader) (*V0Package, error) {
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

	result := &V0Package{}
	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			validationError := result.ValidationError()
			if validationError == nil {
				// bundle is complete
				return result, nil
			}

			return nil, validationError

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
			case isFrameworkV0File(header.Name):
				var f v1alpha1.Framework
				if err = yaml.Unmarshal(bytes, &f); err != nil {
					return nil, errors.Wrapf(err, "unmarshalling %s content", header.Name)
				}
				result.Framework = &f
			case isVersionV0File(header.Name):
				var fv v1alpha1.FrameworkVersion
				if err = yaml.Unmarshal(bytes, &fv); err != nil {
					return nil, errors.Wrapf(err, "unmarshalling %s content", header.Name)
				}
				result.FrameworkVersion = &fv
			case isInstanceV0File(header.Name):
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

func isFrameworkV0File(name string) bool {
	return strings.HasSuffix(name, frameworkV0FileName)
}

func isVersionV0File(name string) bool {
	return strings.HasSuffix(name, versionV0FileName)
}

func isInstanceV0File(name string) bool {
	return strings.HasSuffix(name, instanceV0FileName)
}

func UntarV1Package(r io.Reader) (*V1Package, error) {
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

	result := &V1Package{}
	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			validationError := result.ValidationError()
			if validationError == nil {
				// bundle is complete
				return result, nil
			}

			return nil, validationError

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
			// we don't need to handle folders, files have folder name in their names and that should be enough

		// if it's a file create it
		case tar.TypeReg:
			bytes, err := ioutil.ReadAll(tr)
			if err != nil {
				return nil, errors.Wrapf(err, "while reading file from bundle tarball %s", header.Name)
			}

			switch {
			case isFrameworkV1File(header.Name):
				// TODO
			case isTemplateV1File(header.Name):
				// TODO
			default:
				return nil, fmt.Errorf("unexpected file in the tarball structure %s", header.Name)
			}
		}
	}
}

func isFrameworkV1File(name string) bool {
	return strings.HasSuffix(name, frameworkV1FileName)
}

func isTemplateV1File(name string) bool {
	return strings.HasSuffix(name, templateV1FileName)
}

type V1Package struct {
	// TODO
	// templates ?
	// frameworkFile ?
}

func (p *V1Package) GetFrameworkCRD() (*v1alpha1.Framework, error) { return nil, nil }
func (p *V1Package) GetFrameworkVersionCRD() (*v1alpha1.FrameworkVersion, error) { return nil, nil }
func (p *V1Package) GetInstanceCRD() (*v1alpha1.Instance, error) { return nil, nil }
func (p *V1Package) ValidationError() error { return errors.New("not implemented") }