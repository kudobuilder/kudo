package reader

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

const (
	operatorFileName      = "operator.yaml"
	templateFileNameRegex = "templates/.*.yaml"
	paramsFileName        = "params.yaml"
	APIVersion            = "kudo.dev/v1beta1"
)

func newPackageFiles() packages.Files {
	return packages.Files{
		Templates: make(map[string]string),
	}
}

func parsePackageFile(filePath string, fileBytes []byte, currentPackage *packages.Files) error {
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

func readParametersFile(fileBytes []byte) (packages.ParametersFile, error) {
	paramsFile := packages.ParametersFile{}
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
