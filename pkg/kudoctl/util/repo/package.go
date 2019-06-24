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
	v1FrameworkFileName     = "framework.yaml"
	v1TemplateFileNameRegex = "templates/.*.yaml"
	v1ParamsFileName        = "params.yaml"
)

const apiVersion = "kudo.k8s.io/v1alpha1"

// PackageCRDs is collection of CRDs that are used when installing framework
// during installation, package format is converted to this structure
type PackageCRDs struct {
	Framework        *v1alpha1.Framework
	FrameworkVersion *v1alpha1.FrameworkVersion
	Instance         *v1alpha1.Instance
}

// ReadTarGzPackage reads package from tarball and converts it to the CRD format
func ReadTarGzPackage(r io.Reader) (*PackageCRDs, error) {
	p, err := extractV1Package(r)
	if err != nil {
		return nil, errors.Wrap(err, "while extracting package files")
	}
	return p.getCRDs()
}

// ReadFileSystemPackage reads package from filesystem and converts it to the CRD format
func ReadFileSystemPackage(path string) (*PackageCRDs, error) {
	isTarGz := func() bool {
		if fi, err := os.Stat(path); err == nil {
			return fi.Mode().IsRegular() && strings.HasSuffix(path, ".tar.gz")
		}
		return false
	}

	isFolder := func() bool {
		if fi, err := os.Stat(path); err == nil {
			return fi.IsDir()
		}
		return false
	}

	switch {
	case isTarGz():
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		return ReadTarGzPackage(f)
	case isFolder():
		p, err := fromFolder(path)
		if err != nil {
			return nil, errors.Wrap(err, "while reading package from the file system")
		}
		return p.getCRDs()
	default:
		return nil, fmt.Errorf("unsupported file system format %v. Expect either a tar.gz file or a folder", path)
	}
}

func fromFolder(packagePath string) (*v1Package, error) {
	result := newV1Package()
	err := filepath.Walk(packagePath, func(path string, file os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if file.IsDir() {
			// skip directories
			return nil
		}
		if path == packagePath {
			// skip the root folder, as Walk always starts there
			return nil
		}
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		return readV1PackageFile(path, bytes, &result)
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func readV1PackageFile(filePath string, fileBytes []byte, currentPackage *v1Package) error {
	isV1FrameworkFile := func(name string) bool {
		return strings.HasSuffix(name, v1FrameworkFileName)
	}

	isV1TemplateFile := func(name string) bool {
		matched, _ := regexp.Match(v1TemplateFileNameRegex, []byte(name))
		return matched
	}

	isV1ParametersFile := func(name string) bool {
		return strings.HasSuffix(name, v1ParamsFileName)
	}

	switch {
	case isV1FrameworkFile(filePath):
		if err := yaml.Unmarshal(fileBytes, &currentPackage.Framework); err != nil {
			return errors.Wrap(err, "failed to unmarshal framework")
		}
	case isV1TemplateFile(filePath):
		pathParts := strings.Split(filePath, "templates/")
		name := pathParts[len(pathParts)-1]
		currentPackage.Templates[name] = string(fileBytes)
	case isV1ParametersFile(filePath):
		var params map[string]map[string]string
		if err := yaml.Unmarshal(fileBytes, &params); err != nil {
			return errors.Wrapf(err, "failed to unmarshal parameters file %s", filePath)
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
		currentPackage.Params = paramsStruct
	default:
		return fmt.Errorf("unexpected file when reading package from filesystem %s", filePath)
	}
	return nil
}

func extractV1Package(r io.Reader) (*v1Package, error) {
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

			err = readV1PackageFile(header.Name, bytes, &result)
			if err != nil {
				return nil, err
			}
		}
	}
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

func (p *v1Package) getCRDs() (*PackageCRDs, error) {
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

	return &PackageCRDs{
		Framework:        framework,
		FrameworkVersion: fv,
		Instance:         instance,
	}, nil
}
