package packages

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"regexp"
	"strings"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
	"github.com/kudobuilder/kudo/pkg/util/kudo"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	operatorFileName      = "operator.yaml"
	templateFileNameRegex = "templates/.*.yaml"
	paramsFileName        = "params.yaml"
	APIVersion            = "kudo.dev/v1beta1"
)

// Resources is collection of CRDs that are used when installing operator
// during installation, package format is converted to this structure
type Resources struct {
	Operator        *v1beta1.Operator
	OperatorVersion *v1beta1.OperatorVersion
	Instance        *v1beta1.Instance
}

// PackageFiles represents the raw operator package format the way it is found in the tgz packages
type PackageFiles struct {
	Templates map[string]string
	Operator  *Operator
	Params    []v1beta1.Parameter
}

type ParametersFile struct {
	APIVersion string              `json:"apiVersion,omitempty"`
	Params     []v1beta1.Parameter `json:"parameters"`
}

// Operator is a representation of the KEP-9 Operator YAML
type Operator struct {
	APIVersion        string                  `json:"apiVersion,omitempty"`
	Name              string                  `json:"name"`
	Description       string                  `json:"description,omitempty"`
	Version           string                  `json:"version"`
	AppVersion        string                  `json:"appVersion,omitempty"`
	KUDOVersion       string                  `json:"kudoVersion,omitempty"`
	KubernetesVersion string                  `json:"kubernetesVersion,omitempty"`
	Maintainers       []*v1beta1.Maintainer   `json:"maintainers,omitempty"`
	URL               string                  `json:"url,omitempty"`
	Tasks             []v1beta1.Task          `json:"tasks"`
	Plans             map[string]v1beta1.Plan `json:"plans"`
}

// PackageFilesDigest is a tuple of data used to return the package files AND the digest of a tarball
type PackageFilesDigest struct {
	PkgFiles *PackageFiles
	Digest   string
}

func parsePackageFile(filePath string, fileBytes []byte, currentPackage *PackageFiles) error {
	isOperatorFile := func(name string) bool {
		return strings.HasSuffix(name, operatorFileName)
	}

	isTemplateFile := func(name string) bool {
		matched, err := regexp.Match(templateFileNameRegex, []byte(name))
		if err != nil {
			panic(err)
		}
		return matched
	}

	isParametersFile := func(name string) bool {
		return strings.HasSuffix(name, paramsFileName)
	}

	switch {
	case isOperatorFile(filePath):
		if err := yaml.Unmarshal(fileBytes, &currentPackage.Operator); err != nil {
			return errors.Wrap(err, "failed to unmarshal operator file")
		}
		if currentPackage.Operator.APIVersion == "" {
			currentPackage.Operator.APIVersion = APIVersion
		}
		if currentPackage.Operator.APIVersion != APIVersion {
			return fmt.Errorf("expecting supported API version %s but got %s", APIVersion, currentPackage.Operator.APIVersion)
		}
	case isTemplateFile(filePath):
		pathParts := strings.Split(filePath, "templates/")
		name := pathParts[len(pathParts)-1]
		currentPackage.Templates[name] = string(fileBytes)
	case isParametersFile(filePath):
		paramsFile, err := readParametersFile(fileBytes)
		if err != nil {
			return errors.Wrapf(err, "failed to unmarshal parameters file: %s", filePath)
		}
		currentPackage.Params = make([]v1beta1.Parameter, 0)
		defaultRequired := true
		for _, param := range paramsFile.Params {
			if param.Required == nil {
				// applying default value of required for all params where not specified
				param.Required = &defaultRequired
			}
			currentPackage.Params = append(currentPackage.Params, param)
		}
	default:
		return fmt.Errorf("unexpected file when reading package from filesystem: %s", filePath)
	}
	return nil
}

func readParametersFile(fileBytes []byte) (ParametersFile, error) {
	paramsFile := ParametersFile{}
	if err := yaml.Unmarshal(fileBytes, &paramsFile); err != nil {
		return paramsFile, err
	}
	if paramsFile.APIVersion == "" {
		paramsFile.APIVersion = APIVersion
	}
	if paramsFile.APIVersion != APIVersion {
		return paramsFile, fmt.Errorf("expecting supported API version %s but got %s", APIVersion, paramsFile.APIVersion)
	}

	return paramsFile, nil
}

func newPackageFiles() PackageFiles {
	return PackageFiles{
		Templates: make(map[string]string),
	}
}

func validateTask(t v1beta1.Task, templates map[string]string) []string {
	var resources []string
	switch t.Kind {
	case task.ApplyTaskKind:
		resources = t.Spec.ResourceTaskSpec.Resources
	case task.DeleteTaskKind:
		resources = t.Spec.ResourceTaskSpec.Resources
	case task.DummyTaskKind:
	default:
		log.Printf("no validation for task kind %s implemented", t.Kind)
	}

	var errs []string
	for _, res := range resources {
		if _, ok := templates[res]; !ok {
			errs = append(errs, fmt.Sprintf("task %s missing template: %s", t.Name, res))
		}
	}

	return errs
}

func (p *PackageFiles) getCRDs() (*Resources, error) {
	if p.Operator == nil {
		return nil, errors.New("operator.yaml file is missing")
	}
	if p.Params == nil {
		return nil, errors.New("params.yaml file is missing")
	}
	var errs []string
	for _, tt := range p.Operator.Tasks {
		errs = append(errs, validateTask(tt, p.Templates)...)
	}

	if len(errs) != 0 {
		return nil, errors.New(strings.Join(errs, "\n"))
	}

	operator := &v1beta1.Operator{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Operator",
			APIVersion: APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   p.Operator.Name,
			Labels: map[string]string{"controller-tools.k8s.io": "1.0"},
		},
		Spec: v1beta1.OperatorSpec{
			Description:       p.Operator.Description,
			KudoVersion:       p.Operator.KUDOVersion,
			KubernetesVersion: p.Operator.KubernetesVersion,
			Maintainers:       p.Operator.Maintainers,
			URL:               p.Operator.URL,
		},
		Status: v1beta1.OperatorStatus{},
	}

	fv := &v1beta1.OperatorVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       "OperatorVersion",
			APIVersion: APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s-%s", p.Operator.Name, p.Operator.Version),
			Labels: map[string]string{"controller-tools.k8s.io": "1.0"},
		},
		Spec: v1beta1.OperatorVersionSpec{
			Operator: v1.ObjectReference{
				Name: p.Operator.Name,
				Kind: "Operator",
			},
			Version:        p.Operator.Version,
			Templates:      p.Templates,
			Tasks:          p.Operator.Tasks,
			Parameters:     p.Params,
			Plans:          p.Operator.Plans,
			UpgradableFrom: nil,
		},
		Status: v1beta1.OperatorVersionStatus{},
	}

	instance := &v1beta1.Instance{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Instance",
			APIVersion: APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s-instance", p.Operator.Name),
			Labels: map[string]string{"controller-tools.k8s.io": "1.0", kudo.OperatorLabel: p.Operator.Name},
		},
		Spec: v1beta1.InstanceSpec{
			OperatorVersion: v1.ObjectReference{
				Name: fmt.Sprintf("%s-%s", p.Operator.Name, p.Operator.Version),
			},
		},
		Status: v1beta1.InstanceStatus{},
	}

	return &Resources{
		Operator:        operator,
		OperatorVersion: fv,
		Instance:        instance,
	}, nil
}

// GetFilesDigest maps []string of paths to the [] Operators
func GetFilesDigest(fs afero.Fs, paths []string) []*PackageFilesDigest {
	return mapPaths(fs, paths, pathToOperator)
}

// work of map path, swallows errors to return only packages that are valid
func mapPaths(fs afero.Fs, paths []string, f func(afero.Fs, string) (*PackageFilesDigest, error)) []*PackageFilesDigest {
	ops := make([]*PackageFilesDigest, 0)
	for _, path := range paths {
		op, err := f(fs, path)
		if err != nil {
			fmt.Printf("WARNING: operator: %v is invalid", path)
			continue
		}
		ops = append(ops, op)
	}

	return ops
}

// pathToOperator takes a single path and returns an operator or error
func pathToOperator(fs afero.Fs, path string) (pfd *PackageFilesDigest, err error) {
	reader, err := fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if ferr := reader.Close(); ferr != nil {
			err = ferr
		}
	}()

	digest, err := files.Sha256Sum(reader)
	if err != nil {
		return nil, err
	}
	// restart reading of file after getting digest
	_, err = reader.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	pkg, err := bufferToPackageFiles(bytes.NewBuffer(b))
	pfd = &PackageFilesDigest{
		pkg,
		digest,
	}
	return pfd, err
}

func bufferToPackageFiles(buf *bytes.Buffer) (*PackageFiles, error) {
	b := NewFromBytes(buf)
	pkg, err := b.GetPkgFiles()
	if err != nil {
		return nil, err
	}
	return pkg, nil
}
