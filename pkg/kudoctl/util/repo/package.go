package repo

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/bundle"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"regexp"
	"sigs.k8s.io/yaml"
	"strings"
)

const (
	frameworkV0FileName = "-framework.yaml"
	versionV0FileName   = "-frameworkversion.yaml"
	instanceV0FileName  = "-instance.yaml"

	frameworkV1FileName = "framework.yaml"
	templatesV1Folder = "templates"
	templateV1FileNameRegex = "templates/.*.yaml"
	paramsV1FileName = "params.yaml"
)

const apiVersion = "kudo.k8s.io/v1alpha1"

type InstallCRDs struct {
	Framework        *v1alpha1.Framework
	FrameworkVersion *v1alpha1.FrameworkVersion
	Instance         *v1alpha1.Instance
}

type FrameworkPackage interface {
	GetInstallCRDs() (*InstallCRDs, error)
}

type V0Package struct {
	Framework        *v1alpha1.Framework
	FrameworkVersion *v1alpha1.FrameworkVersion
	Instance         *v1alpha1.Instance
}

func (p *V0Package) GetInstallCRDs() (*InstallCRDs, error) {
	validationError := p.validate()
	if validationError == nil {
		return &InstallCRDs{
			Framework: p.Framework,
			FrameworkVersion: p.FrameworkVersion,
			Instance: p.Instance,
		}, nil
	}
	return nil, validationError
}
func (p *V0Package) validate() error {
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

func ConvertV0Package(r io.Reader) (*InstallCRDs, error) {
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
			return result.GetInstallCRDs()

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

func ReadTarballPackage(r io.Reader) (*InstallCRDs, error) {
	p, err := untarV1Package(r)
	if err != nil {
		return nil, errors.Wrap(err, "while untarring package")
	}
	return p.GetInstallCRDs()
}

func ReadFileSystemPackage(path string) (*InstallCRDs, error) {
	p, err := fromFilesystem(path)
	if err != nil {
		return nil, errors.Wrap(err, "while reading package from filesystem")
	}
	return p.GetInstallCRDs()
}

func fromFilesystem(packagePath string) (*V1Package, error) {
	result := NewV1Package()
	err := filepath.Walk(packagePath, func(path string, info os.FileInfo, err error) error {
		relativePath := strings.TrimPrefix(path, packagePath)
			if path == packagePath {
				// skip the root folder, as Walk always starts there
				return nil
			}
			bytes, err := ioutil.ReadFile(path)
			switch {
			case isFrameworkV1File(info.Name()):
			var bf bundle.Framework

			if err = yaml.Unmarshal(bytes, &bf); err != nil {
				return errors.Wrap(err, "cannot unmarshal framework")
			}
			result.Framework = &bf
		case info.Name() == templatesV1Folder && info.IsDir():
			// skip the folder itself, wait until we recursively start going into the template files
			return nil
		case isTemplateV1File(relativePath):
			name := strings.TrimPrefix(relativePath, "/templates/")
			result.Templates[name] = string(bytes)
		case isParametersV1File(info.Name()):
			if err = yaml.Unmarshal(bytes, &result.Params); err != nil {
				return errors.Wrapf(err, "unmarshalling %s content", info.Name())
			}
		default:
			return fmt.Errorf("unexpected file in the fileststem structure %s", info.Name())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func untarV1Package(r io.Reader) (*V1Package, error) {
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

	result := NewV1Package()
	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return &result, nil

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
				var bf bundle.Framework

				if err = yaml.Unmarshal(bytes, &bf); err != nil {
					return nil, errors.Wrap(err, "cannot unmarshal framework")
				}
				result.Framework = &bf
			case isTemplateV1File(header.Name):
				name := strings.TrimPrefix(header.Name, "templates/")
				result.Templates[name] = string(bytes)
			case isParametersV1File(header.Name):
				if err = yaml.Unmarshal(bytes, &result.Params); err != nil {
					return nil, errors.Wrapf(err, "unmarshalling %s content", header.Name)
				}
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
	matched, _ := regexp.Match(templateV1FileNameRegex, []byte(name))
	return matched
}

func isParametersV1File(name string) bool {
	return strings.HasSuffix(name, paramsV1FileName)
}

type V1Package struct {
	Templates map[string]string
	Framework *bundle.Framework
	Params map[string]map[string]string
}

func NewV1Package() V1Package {
	return V1Package{
		Templates: make(map[string]string),
	}
}

func (p *V1Package) GetInstallCRDs() (*InstallCRDs, error) {
	if p.Framework == nil {
		return nil, errors.New("framework.yaml file is missing")
	}
	if p.Params == nil {
		return nil, errors.New("params.yaml file is missing")
	}
	var errs []string
	for k, v := range p.Framework.Tasks {
		for _, res := range v.Resources {
			if _, ok := p.Templates[res]; !ok {
				errs = append(errs, fmt.Sprintf("task %s missing template: %s", k, res))
			}
		}
	}

	if len(errs) != 0 {
		return nil, errors.New(strings.Join(errs, "\n"))
	}

	framework := &v1alpha1.Framework{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Framework",
			APIVersion: apiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: p.Framework.Name,
			Labels: map[string]string{ "controller-tools.k8s.io": "1.0" },
		},
		Spec: v1alpha1.FrameworkSpec{
			Description:       p.Framework.Description,
			KudoVersion:       p.Framework.KUDOVersion,
			KubernetesVersion: p.Framework.KubernetesVersion,
			Maintainers:       p.Framework.Maintainers,
			URL:               p.Framework.URL,
		},
		Status: v1alpha1.FrameworkStatus{},
	}

	params := make([]v1alpha1.Parameter, 0)
	for paramName, param := range p.Params {
		result := v1alpha1.Parameter{
			Name: paramName,
			Description: param["description"],
			Default: param["default"],
		}
		params = append(params, result)
	}

	fv := &v1alpha1.FrameworkVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       "FrameworkVersion",
			APIVersion: apiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", p.Framework.Name, p.Framework.Version),
			Namespace: "default",
			Labels: map[string]string{ "controller-tools.k8s.io": "1.0" },
		},
		Spec: v1alpha1.FrameworkVersionSpec{
			Framework: v1.ObjectReference{
				Name: p.Framework.Name,
				Kind: "Framework",
			},
			Version:        p.Framework.Version,
			Templates:      p.Templates,
			Tasks:          p.Framework.Tasks,
			Parameters:     params,
			Plans:          p.Framework.Plans,
			Dependencies:   p.Framework.Dependencies,
			UpgradableFrom: nil,
		},
		Status: v1alpha1.FrameworkVersionStatus{},
	}

	instance := &v1alpha1.Instance{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Instance",
			APIVersion: apiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", p.Framework.Name, p.Framework.Version),
			Labels: map[string]string{ "controller-tools.k8s.io": "1.0", "framework": "zookeeper" },
		},
		Spec: v1alpha1.InstanceSpec{
			FrameworkVersion: v1.ObjectReference{
				Name: fmt.Sprintf("%s-%s", p.Framework.Name, p.Framework.Version),
				Namespace: "default",
			},
		},
		Status: v1alpha1.InstanceStatus{},
	}

	return &InstallCRDs{
		Framework: framework,
		FrameworkVersion: fv,
		Instance: instance,
	}, nil
}
