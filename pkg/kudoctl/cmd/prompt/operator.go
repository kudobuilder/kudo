package prompt

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/generate"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

// ForOperator prompts to gather details for a new operator
func ForOperator(fs afero.Fs, pathDefault string, overwrite bool, operatorDefault packages.OperatorFile) (*packages.OperatorFile, string, error) {

	nameValid := func(input string) error {
		if len(input) < 3 {
			return errors.New("Operator name must have more than 3 characters")
		}
		return nil
	}

	name, err := WithValidator("Operator Name", operatorDefault.Name, nameValid)
	if err != nil {
		return nil, "", err
	}

	pathValid := func(input string) error {
		if len(input) < 1 {
			return errors.New("Operator directory must have more than 1 character")
		}
		return generate.CanGenerateOperator(fs, input, overwrite)
	}

	path, err := WithValidator("Operator directory", pathDefault, pathValid)
	if err != nil {
		return nil, "", err
	}

	versionValid := func(input string) error {
		if len(input) < 1 {
			return errors.New("Operator version is required in semver format")
		}
		_, err := semver.NewVersion(input)
		return err
	}
	opVersion, err := WithValidator("Operator Version", operatorDefault.Version, versionValid)
	if err != nil {
		return nil, "", err
	}

	appVersion, err := WithDefault("Application Version", "")
	if err != nil {
		return nil, "", err
	}

	kudoVersion, err := WithDefault("Required KUDO Version", operatorDefault.KUDOVersion)
	if err != nil {
		return nil, "", err
	}

	url, err := WithDefault("Project URL", "")
	if err != nil {
		return nil, "", err
	}

	op := packages.OperatorFile{
		Name:        name,
		APIVersion:  operatorDefault.APIVersion,
		Version:     opVersion,
		AppVersion:  appVersion,
		KUDOVersion: kudoVersion,
		URL:         url,
	}
	return &op, path, nil
}

// ForMaintainer prompts to gather information to add an operator maintainer
func ForMaintainer() (*v1beta1.Maintainer, error) {

	nameValid := func(input string) error {
		if len(input) < 1 {
			return errors.New("Maintainer name must be > than 1 character")
		}
		return nil
	}

	name, err := WithValidator("Maintainer Name", "", nameValid)
	if err != nil {
		return nil, err
	}

	emailValid := func(input string) error {
		if !validEmail(input) {
			return errors.New("maintainer email must be valid email address")
		}
		return nil
	}

	email, err := WithValidator("Maintainer Email", "", emailValid)
	if err != nil {
		return nil, err
	}

	return &v1beta1.Maintainer{Name: name, Email: email}, nil
}

func validEmail(email string) bool {
	re := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	return re.MatchString(email)
}

// ForParameter prompts to gather information to add an operator parameter
func ForParameter(planNames []string) (*v1beta1.Parameter, error) {
	nameValid := func(input string) error {
		if len(input) < 1 {
			return errors.New("Parameter name must be > than 1 character")
		}
		return nil
	}
	name, err := WithValidator("Parameter Name", "", nameValid)
	if err != nil {
		return nil, err
	}

	value, err := WithDefault("Default Value", "")
	if err != nil {
		return nil, err
	}

	displayName, err := WithDefault("Display Name", "")
	if err != nil {
		return nil, err
	}

	// building param
	parameter := v1beta1.Parameter{
		DisplayName: displayName,
		Name:        name,
	}
	if value != "" {
		parameter.Default = &value
	}

	desc, err := WithDefault("Description", "")
	if err != nil {
		return nil, err
	}
	if desc != "" {
		parameter.Description = desc
	}

	// order determines the default ("false" is preferred)
	requiredValues := []string{"false", "true"}
	required, err := WithOptions("Required", requiredValues, true)
	if err != nil {
		return nil, err
	}
	if required == "true" {
		t := true
		parameter.Required = &t
	}

	//PlanNameList
	var trigger string
	if len(planNames) == 0 {
		trigger, err = WithDefault("Trigger Plan", "")
	} else {
		trigger, err = WithOptions("Trigger Plan", planNames, true)
	}
	if err != nil {
		return nil, err
	}
	if trigger != "" {
		parameter.Trigger = trigger
	}

	return &parameter, nil
}

// ForTask prompts to gather information to add new task to operator
func ForTask(existingTasks []v1beta1.Task) (*v1beta1.Task, error) {
	nameValid := func(input string) error {
		if len(input) < 1 {
			return errors.New("Task name must be > than 1 character")
		}
		if taskExists(input, existingTasks) {
			return errors.New("Task name must be unique")
		}
		return nil
	}
	name, err := WithValidator("Task Name", "", nameValid)
	if err != nil {
		return nil, err
	}

	kind, err := WithOptions("Task Kind", generate.TaskKinds(), false)
	if err != nil {
		return nil, err
	}

	var again bool
	resources := []string{}
	for {
		resource, err := WithDefault("Task Resource", "")
		if err != nil {
			return nil, err
		}
		resources = append(resources, ensureFileExtension(resource, "yaml"))

		again = Confirm("Add another resource")
		if !again {
			break
		}
	}

	//TODO (kensipe): lets add pipe tasks!
	spec := v1beta1.TaskSpec{
		ResourceTaskSpec: v1beta1.ResourceTaskSpec{Resources: resources},
	}

	task := v1beta1.Task{
		Name: name,
		Kind: kind,
		Spec: spec,
	}

	return &task, nil
}

func taskExists(name string, existingTasks []v1beta1.Task) bool {
	for _, task := range existingTasks {
		if task.Name == name {
			return true
		}
	}
	return false
}

func ensureFileExtension(fname, ext string) string {
	if strings.Contains(fname, ".") {
		return fname
	}
	return fmt.Sprintf("%s.%s", fname, ext)
}
