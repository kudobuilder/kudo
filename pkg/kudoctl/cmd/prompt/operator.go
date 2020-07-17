package prompt

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/spf13/afero"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine/task"
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
	opVersion, err := WithValidator("Operator Version", operatorDefault.OperatorVersion, versionValid)
	if err != nil {
		return nil, "", err
	}

	optionalVersionValid := func(input string) error {
		if len(input) < 1 {
			return nil
		}
		_, err := semver.NewVersion(input)
		return err
	}
	appVersion, err := WithValidator("Application Version", "", optionalVersionValid)
	if err != nil {
		return nil, "", err
	}

	kudoVersion, err := WithDefault("Required KUDO Version", operatorDefault.KUDOVersion)
	if err != nil {
		return nil, "", err
	}

	kubernetesVersion, err := WithDefault("Required Kubernetes Version", operatorDefault.KubernetesVersion)
	if err != nil {
		return nil, "", err
	}

	url, err := WithDefault("Project URL", "")
	if err != nil {
		return nil, "", err
	}

	op := packages.OperatorFile{
		Name:              name,
		APIVersion:        operatorDefault.APIVersion,
		OperatorVersion:   opVersion,
		AppVersion:        appVersion,
		KUDOVersion:       kudoVersion,
		KubernetesVersion: kubernetesVersion,
		URL:               url,
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
func ForParameter(planNames []string, paramNameList []string) (*packages.Parameter, error) {
	nameValid := func(input string) error {
		if len(input) < 1 {
			return errors.New("Parameter name must be > than 1 character")
		}
		if inArray(input, paramNameList) {
			return errors.New("Parameter name must be unique")
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
	parameter := packages.Parameter{
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
	required, err := WithOptions("Required", requiredValues, "")
	if err != nil {
		return nil, err
	}
	if required == "true" {
		t := true
		parameter.Required = &t
	}

	//PlanNameList
	if Confirm("Add Trigger Plan (defaults to deploy)") {
		var trigger string
		if len(planNames) == 0 {
			trigger, err = WithDefault("Trigger Plan", "")
		} else {
			trigger, err = WithOptions("Trigger Plan", planNames, "New plan name to trigger")
		}
		if err != nil {
			return nil, err
		}
		if trigger != "" {
			parameter.Trigger = trigger
		}
	}

	return &parameter, nil
}

func ForTaskName(existingTasks []v1beta1.Task) (string, error) {
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
		return "", err
	}
	return name, nil
}

// ForTask prompts to gather information to add new task to operator
func ForTask(name string) (*v1beta1.Task, error) {

	kind, err := WithOptions("Task Kind", generate.TaskKinds(), "")
	if err != nil {
		return nil, err
	}
	spec := v1beta1.TaskSpec{}

	switch kind {
	case task.ApplyTaskKind:
		fallthrough
	case task.DeleteTaskKind:
		var again bool
		resources := []string{}
		for {
			resource, err := WithDefault("Task Resource", "")
			if err != nil {
				return nil, err
			}
			resources = append(resources, ensureFileExtension(resource, "yaml"))

			again = Confirm("Add another Resource")
			if !again {
				break
			}
		}
		spec.ResourceTaskSpec = v1beta1.ResourceTaskSpec{Resources: resources}

	case task.PipeTaskKind:
		pod, err := WithDefault("Pipe Pod File", "")
		if err != nil {
			return nil, err
		}
		var again bool
		pipes := []v1beta1.PipeSpec{}
		for {
			file, err := WithDefault("Pipe File (internal to pod)", "")
			if err != nil {
				return nil, err
			}
			kind, err := WithOptions("Pipe Kind", []string{"ConfigMap", "Secret"}, "")
			if err != nil {
				return nil, err
			}
			key, err := WithDefault("Pipe Kind Key", "")
			if err != nil {
				return nil, err
			}
			pipes = append(pipes, v1beta1.PipeSpec{
				File: file,
				Kind: kind,
				Key:  key,
			})
			again = Confirm("Add another pipe")
			if !again {
				break
			}
		}
		spec.PipeTaskSpec = v1beta1.PipeTaskSpec{
			Pod:  ensureFileExtension(pod, "yaml"),
			Pipe: pipes,
		}
	}

	t := v1beta1.Task{
		Name: name,
		Kind: kind,
		Spec: spec,
	}

	return &t, nil
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

func ForPlan(planNames []string, tasks []v1beta1.Task, fs afero.Fs, path string, createTaskFun func(fs afero.Fs, path string, taskName string) error) (string, *v1beta1.Plan, error) {

	// initialized to all TaskNames... tasks can be added in the process of creating a plan which will be
	// added to this list in the process.
	allTaskNames := []string{}
	for _, task := range tasks {
		allTaskNames = append(allTaskNames, task.Name)
	}

	nameValid := func(input string) error {
		if len(input) < 1 {
			return errors.New("Plan name must be > than 1 character")
		}
		if inArray(input, planNames) {
			return errors.New("Plan name must be unique")
		}
		return nil
	}
	var defaultPlanName, defaultPhaseName, defaultStepName, defaultTaskName string
	if len(planNames) == 0 {
		defaultPlanName = "deploy"
		defaultPhaseName = defaultPlanName
		defaultStepName = defaultPlanName
		defaultTaskName = defaultPlanName
	}

	name, err := WithValidator("Plan Name", defaultPlanName, nameValid)
	if err != nil {
		return "", nil, err
	}

	planStrat, err := WithOptions("Plan strategy for phase", []string{string(v1beta1.Serial), string(v1beta1.Parallel)}, "")
	if err != nil {
		return "", nil, err
	}
	plan := v1beta1.Plan{
		Strategy: v1beta1.Ordering(planStrat),
	}

	// setting up for array of phases in a plan
	index := 1
	anotherPhase := false
	phaseNames := []string{}
	phases := []v1beta1.Phase{}
	for {
		phase, err := forPhase(phaseNames, index, defaultPhaseName)
		if err != nil {
			return "", nil, err
		}

		// setting up for array of steps in a phase
		stepIndex := 1
		anotherStep := false
		var stepNames []string
		var steps []v1beta1.Step
		for {

			step, err := forStep(stepNames, stepIndex, defaultStepName)
			if err != nil {
				return "", nil, err
			}

			stepIndex++
			stepNames = append(stepNames, step.Name)

			// setting up for array of tasks in a step
			var stepTaskNames []string
			taskIndex := 1
			anotherTask := false
			for {
				taskName, err := forStepTaskName(allTaskNames, stepTaskNames, taskIndex, step.Name, defaultTaskName)
				if err != nil {
					return "", nil, err
				}
				if !inArray(taskName, allTaskNames) {
					if Confirm("Create Task Now") {
						err = createTaskFun(fs, path, taskName)
						if err != nil {
							return "", nil, err
						}
					}
					allTaskNames = append(allTaskNames, taskName)
				}
				stepTaskNames = append(stepTaskNames, taskName)
				taskIndex++
				defaultTaskName = ""
				anotherTask = Confirm("Add another Task")
				if !anotherTask {
					break
				}
			}

			step.Tasks = stepTaskNames
			steps = append(steps, *step)
			defaultStepName = ""
			anotherStep = Confirm("Add another Step")
			if !anotherStep {
				break
			}
		}
		phase.Steps = steps

		phases = append(phases, *phase)
		index++
		defaultPhaseName = ""
		anotherPhase = Confirm("Add another Phase")
		if !anotherPhase {
			break
		}
	}
	plan.Phases = phases

	return name, &plan, nil

}

func forStepTaskName(allTaskNames []string, stepTaskNames []string, taskIndex int, name string, defaultTaskName string) (taskName string, err error) {
	// reduce options of tasks to those not already for this step
	taskNameOptions := subtract(allTaskNames, stepTaskNames)
	// if there are no tasks OR if we are using all tasks that are defined
	if len(taskNameOptions) == 0 {
		// no list if there is nothing in the list
		taskName, err = WithDefault(fmt.Sprintf("Task Name %v for Step %q", taskIndex, name), defaultTaskName)
	} else {
		taskName, err = WithOptions(fmt.Sprintf("Task Name %v for Step %q", taskIndex, name), taskNameOptions, "Add New Task")
	}
	return taskName, err
}

func subtract(allTasksNames []string, currentStepTaskNames []string) (result []string) {
	for _, name := range allTasksNames {
		if !inArray(name, currentStepTaskNames) {
			result = append(result, name)
		}
	}
	return result
}

func forStep(stepNames []string, stepIndex int, defaultStepName string) (*v1beta1.Step, error) {
	stepNameValid := func(input string) error {
		if len(input) < 1 {
			return errors.New("Step name must be > than 1 character")
		}
		if inArray(input, stepNames) {
			return errors.New("Step name must be unique in a Phase")
		}
		return nil
	}
	stepName, err := WithValidator(fmt.Sprintf("Step %v name", stepIndex), defaultStepName, stepNameValid)
	if err != nil {
		return nil, err
	}
	step := v1beta1.Step{Name: stepName}
	return &step, nil
}

func forPhase(phaseNames []string, index int, defaultPhaseName string) (*v1beta1.Phase, error) {
	pnameValid := func(input string) error {
		if len(input) < 1 {
			return errors.New("Phase name must be > than 1 character")
		}
		if inArray(input, phaseNames) {
			return errors.New("Phase name must be unique in plan")
		}
		return nil
	}
	pname, err := WithValidator(fmt.Sprintf("Phase %v name", index), defaultPhaseName, pnameValid)
	if err != nil {
		return nil, err
	}
	phaseStrat, err := WithOptions("Phase strategy for steps", []string{string(v1beta1.Serial), string(v1beta1.Parallel)}, "")
	if err != nil {
		return nil, err
	}
	phase := v1beta1.Phase{
		Name:     pname,
		Strategy: v1beta1.Ordering(phaseStrat),
	}
	return &phase, nil
}

func inArray(input string, values []string) bool {
	for _, name := range values {
		if input == name {
			return true
		}
	}
	return false
}
