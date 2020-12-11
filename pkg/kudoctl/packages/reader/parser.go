package reader

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

const (
	OperatorFileName = "operator.yaml"
	ParamsFileName   = "params.yaml"
	templateBase     = "templates"
	templateFileName = ".*\\.yaml"
	APIVersion       = "kudo.dev/v1beta1"
)

func newPackageFiles() packages.Files {
	return packages.Files{
		Templates: make(map[string]string),
	}
}

// parsePackageFile parses the passed file into the correct type and adds it to the currentPackage
// The filePath needs to be relative to the package root
func parsePackageFile(filePath string, fileBytes []byte, currentPackage *packages.Files) error {
	isOperatorFile := func(name string) bool {
		dir, file := filepath.Split(name)
		base := filepath.Base(dir)

		return base == "." && file == OperatorFileName
	}

	isTemplateFile := func(name string) bool {
		name = filepath.Clean(name)

		dir, file := filepath.Split(name)
		if !strings.HasPrefix(dir, templateBase) {
			return false
		}

		match, err := regexp.MatchString(templateFileName, file)
		if err != nil {
			clog.Printf("Failed to parse template file %s, err: %v", name, err)
			os.Exit(1)
		}

		return match
	}

	isParametersFile := func(name string) bool {
		dir, file := filepath.Split(name)
		base := filepath.Base(dir)

		return base == "." && file == ParamsFileName
	}

	switch {
	case isOperatorFile(filePath):
		if err := yaml.Unmarshal(fileBytes, &currentPackage.Operator); err != nil {
			return fmt.Errorf("failed to unmarshal operator file: %w", err)
		}
		if currentPackage.Operator == nil {
			return fmt.Errorf("failed to parse yaml into valid operator %s", filePath)
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
			return fmt.Errorf("failed to unmarshal parameters file: %s: %w", filePath, err)
		}
		defaultRequired := true
		defaultImmutable := false
		for i := 0; i < len(paramsFile.Parameters); i++ {
			p := &paramsFile.Parameters[i]
			if p.Required == nil {
				// applying default value of required for all params where not specified
				p.Required = &defaultRequired
			}
			if p.Immutable == nil {
				p.Immutable = &defaultImmutable
			}
		}
		currentPackage.Params = &paramsFile
	default:
		// we ignore unexpected files as they might belong to a dependency operator
		return nil
	}
	return nil
}

func readParametersFile(fileBytes []byte) (packages.ParamsFile, error) {
	paramsFile := packages.ParamsFile{}
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
