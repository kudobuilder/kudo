package repo

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/bundle"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	frameworkV1FileName     = "framework.yaml"
	templatesV1Folder       = "templates"
	templateV1FileNameRegex = "templates/.*.yaml"
	paramsV1FileName        = "params.yaml"
)

const apiVersion = "kudo.k8s.io/v1alpha1"

// InstallCRDs is collection of CRDs that are used when installing framework
// during installation, package format is converted to this structure
type InstallCRDs struct {
	Framework        *v1alpha1.Framework
	FrameworkVersion *v1alpha1.FrameworkVersion
	Instance         *v1alpha1.Instance
}

// ReadTarballPackage reads package from tarball and converts it to the CRD format
func ReadTarballPackage(r io.Reader) (*InstallCRDs, error) {
	p, err := untarV1Package(r)
	if err != nil {
		return nil, errors.Wrap(err, "while untarring package")
	}
	return p.getInstallCRDs()
}

// ReadFileSystemPackage reads package from filesystem and converts it to the CRD format
func ReadFileSystemPackage(path string) (*InstallCRDs, error) {
	p, err := fromFilesystem(path)
	if err != nil {
		return nil, errors.Wrap(err, "while reading package from filesystem")
	}
	return p.getInstallCRDs()
}

func fromFilesystem(packagePath string) (*v1Package, error) {
	result := newV1Package()
	err := filepath.Walk(packagePath, func(path string, file os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if file.IsDir() {
			// skip directories
			return nil
		}
		relativePath := strings.TrimPrefix(path, packagePath)
		if path == packagePath {
			// skip the root folder, as Walk always starts there
			return nil
		}
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		switch {
		case isFrameworkV1File(file.Name()):
			var bf bundle.Framework

			if err = yaml.Unmarshal(bytes, &bf); err != nil {
				return errors.Wrap(err, "cannot unmarshal framework")
			}
			result.Framework = &bf
		case file.Name() == templatesV1Folder && file.IsDir():
			// skip the folder itself, wait until we recursively start going into the template files
			return nil
		case isTemplateV1File(relativePath):
			name := strings.TrimPrefix(relativePath, "/templates/")
			result.Templates[name] = string(bytes)
		case isParametersV1File(file.Name()):
			var params map[string]map[string]string
			if err = yaml.Unmarshal(bytes, &params); err != nil {
				return errors.Wrapf(err, "unmarshalling %s content", file.Name())
			}
			paramsStruct := make([]v1alpha1.Parameter, 0)
			for paramName, param := range params {
				r := v1alpha1.Parameter{
					Name:        paramName,
					Description: param["description"],
					Default:     param["default"],
				}
				paramsStruct = append(paramsStruct, r)
			}
			result.Params = paramsStruct
		default:
			return fmt.Errorf("unexpected file when reading package from filesystem %s", file.Name())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func untarV1Package(r io.Reader) (*v1Package, error) {
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

	result := newV1Package()
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
				name := strings.TrimPrefix(header.Name, "/templates/")
				result.Templates[name] = string(bytes)
			case isParametersV1File(header.Name):
				var params map[string]map[string]string
				if err = yaml.Unmarshal(bytes, &params); err != nil {
					return nil, errors.Wrapf(err, "unmarshalling %s content", header.Name)
				}
				paramsStruct := make([]v1alpha1.Parameter, 0)
				for paramName, param := range params {
					r := v1alpha1.Parameter{
						Name:        paramName,
						Description: param["description"],
						Default:     param["default"],
					}
					paramsStruct = append(paramsStruct, r)
				}
				result.Params = paramsStruct
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

type v1Package struct {
	Templates map[string]string
	Framework *bundle.Framework
	Params    []v1alpha1.Parameter
}

func newV1Package() v1Package {
	return v1Package{
		Templates: make(map[string]string),
	}
}

func (p *v1Package) getInstallCRDs() (*InstallCRDs, error) {
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
			Name:   p.Framework.Name,
			Labels: map[string]string{"controller-tools.k8s.io": "1.0"},
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

	fv := &v1alpha1.FrameworkVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       "FrameworkVersion",
			APIVersion: apiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", p.Framework.Name, p.Framework.Version),
			Namespace: "default",
			Labels:    map[string]string{"controller-tools.k8s.io": "1.0"},
		},
		Spec: v1alpha1.FrameworkVersionSpec{
			Framework: v1.ObjectReference{
				Name: p.Framework.Name,
				Kind: "Framework",
			},
			Version:        p.Framework.Version,
			Templates:      p.Templates,
			Tasks:          p.Framework.Tasks,
			Parameters:     p.Params,
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
			Name:   fmt.Sprintf("%s-%s", p.Framework.Name, p.Framework.Version),
			Labels: map[string]string{"controller-tools.k8s.io": "1.0", "framework": "zookeeper"},
		},
		Spec: v1alpha1.InstanceSpec{
			FrameworkVersion: v1.ObjectReference{
				Name:      fmt.Sprintf("%s-%s", p.Framework.Name, p.Framework.Version),
				Namespace: "default",
			},
		},
		Status: v1alpha1.InstanceStatus{},
	}

	return &InstallCRDs{
		Framework:        framework,
		FrameworkVersion: fv,
		Instance:         instance,
	}, nil
}
