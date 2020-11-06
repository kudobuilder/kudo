package task

import (
	"errors"
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/engine/renderer"
)

// Context is a engine.task execution context containing k8s client, templates parameters etc.
type Context struct {
	Client     client.Client
	Discovery  discovery.CachedDiscoveryInterface
	Config     *rest.Config
	Scheme     *runtime.Scheme
	Enhancer   renderer.Enhancer
	Meta       renderer.Metadata
	Templates  map[string]string      // Raw templates
	Parameters map[string]interface{} // Instance and OperatorVersion parameters merged
	Pipes      map[string]string      // Pipe artifacts
}

// Tasker is an interface that represents any runnable task for an operator. This method is treated
// as idempotent and will be called multiple times during the life-cycle of the plan execution.
// Method returns a boolean, signalizing that the task has finished successfully, and an error.
// An error can wrap the ErrFatalExecution for errors that should not be retried e.g. failed template
// rendering. This will result in a kudoapi.ExecutionFatalError in the plan execution status. A normal
// error e.g. failure during accessing the API server will be treated  as "transient", meaning plan
// execution engine  can retry it next time. Only a (true, nil) return value will be treated as successful
// task execution.
type Tasker interface {
	Run(ctx Context) (bool, error)
}

// Available tasks kinds
const (
	ApplyTaskKind        = "Apply"
	DeleteTaskKind       = "Delete"
	DummyTaskKind        = "Dummy"
	PipeTaskKind         = "Pipe"
	ToggleTaskKind       = "Toggle"
	KudoOperatorTaskKind = "KudoOperator"
)

var (
	taskRenderingError      = "TaskRenderingError"
	taskEnhancementError    = "TaskEnhancementError"
	dummyTaskError          = "DummyTaskError"
	resourceUnmarshalError  = "ResourceUnmarshalError"
	resourceValidationError = "ResourceValidationError"
	failedTerminalState     = "FailedTerminalStateError"
)

// Build factory method takes an kudoapi.Task and returns a corresponding Tasker object
func Build(task *kudoapi.Task) (Tasker, error) {
	switch task.Kind {
	case ApplyTaskKind:
		return newApply(task)
	case DeleteTaskKind:
		return newDelete(task)
	case DummyTaskKind:
		return newDummy(task)
	case PipeTaskKind:
		return newPipe(task)
	case ToggleTaskKind:
		return newToggle(task)
	case KudoOperatorTaskKind:
		return newKudoOperator(task)
	default:
		return nil, fmt.Errorf("unknown task kind %s", task.Kind)
	}
}

func newApply(task *kudoapi.Task) (Tasker, error) {
	// validate ApplyTask
	if len(task.Spec.ResourceTaskSpec.Resources) == 0 {
		return nil, fmt.Errorf("task validation error: apply task '%s' has an empty resource list. if that's what you need, use a Dummy task instead", task.Name)
	}

	return ApplyTask{
		Name:      task.Name,
		Resources: task.Spec.ResourceTaskSpec.Resources,
	}, nil
}

func newDelete(task *kudoapi.Task) (Tasker, error) {
	// validate DeleteTask
	if len(task.Spec.ResourceTaskSpec.Resources) == 0 {
		return nil, fmt.Errorf("task validation error: delete task '%s' has an empty resource list. if that's what you need, use a Dummy task instead", task.Name)
	}

	return DeleteTask{
		Name:      task.Name,
		Resources: task.Spec.ResourceTaskSpec.Resources,
	}, nil
}

func newDummy(task *kudoapi.Task) (Tasker, error) {
	return DummyTask{
		Name:    task.Name,
		WantErr: task.Spec.DummyTaskSpec.WantErr,
		Fatal:   task.Spec.DummyTaskSpec.Fatal,
		Done:    task.Spec.DummyTaskSpec.Done,
	}, nil
}

func newPipe(task *kudoapi.Task) (Tasker, error) {
	// validate PipeTask
	if len(task.Spec.PipeTaskSpec.Pipe) == 0 {
		return nil, errors.New("task validation error: pipe task has an empty pipe files list")
	}

	var pipeFiles []PipeFile
	for _, pp := range task.Spec.PipeTaskSpec.Pipe {
		pf := PipeFile{File: pp.File, EnvFile: pp.EnvFile, Kind: PipeFileKind(pp.Kind), Key: pp.Key}
		// validate pipe file
		if err := validPipeFile(pf); err != nil {
			return nil, err
		}
		pipeFiles = append(pipeFiles, pf)
	}

	return PipeTask{
		Name:      task.Name,
		Pod:       task.Spec.PipeTaskSpec.Pod,
		PipeFiles: pipeFiles,
	}, nil
}

func newToggle(task *kudoapi.Task) (Tasker, error) {
	// validate if resources are present
	if len(task.Spec.Resources) == 0 {
		return nil, errors.New("task validation error: toggle task has an empty resource list. if that's what you need, use a Dummy task instead")
	}
	// validate if the parameter is present
	if task.Spec.ToggleTaskSpec.Parameter == "" {
		return nil, errors.New("task validation error: Missing parameter to evaluate the Toggle Task")
	}
	return ToggleTask{
		Name:      task.Name,
		Resources: task.Spec.ResourceTaskSpec.Resources,
		Parameter: task.Spec.ToggleTaskSpec.Parameter,
	}, nil
}

var (
	pipeFileKeyRe = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`) // a-z, A-Z, 0-9, _ and - are allowed
)

func validPipeFile(pf PipeFile) error {
	fl := pf.File != ""
	efl := pf.EnvFile != ""
	if fl == efl {
		return fmt.Errorf("task validation error: pipe file %v must have either 'file' or 'envFile' field set but not both", pf)
	}

	if pf.Kind != PipeFileKindSecret && pf.Kind != PipeFileKindConfigMap {
		return fmt.Errorf("task validation error: invalid pipe kind (must be Secret or ConfigMap): %v", pf)
	}
	if !pipeFileKeyRe.MatchString(pf.Key) {
		return fmt.Errorf("task validation error: invalid pipe key (only letters, numbers and _ and - are allowed): %v", pf)
	}
	return nil
}

// fatalExecutionError is a helper method providing proper wrapping an message for the ExecutionError
func fatalExecutionError(cause error, eventName string, meta renderer.Metadata) engine.ExecutionError {
	return engine.ExecutionError{
		Err: fmt.Errorf("%w%s/%s failed in %s.%s.%s.%s: %v",
			engine.ErrFatalExecution,
			meta.InstanceNamespace,
			meta.InstanceName,
			meta.PlanName,
			meta.PhaseName,
			meta.StepName,
			meta.TaskName,
			cause),
		EventName: eventName,
	}
}

func newKudoOperator(task *kudoapi.Task) (Tasker, error) {
	// validate KudoOperatorTask
	if task.Spec.KudoOperatorTaskSpec.Package == "" {
		return nil, fmt.Errorf("task validation error: kudo operator task '%s' has an empty package name", task.Name)
	}

	if task.Spec.KudoOperatorTaskSpec.OperatorVersion == "" {
		return nil, fmt.Errorf("task validation error: kudo operator task '%s' has an empty operatorVersion", task.Name)
	}

	return KudoOperatorTask{
		Name:            task.Name,
		OperatorName:    task.Spec.KudoOperatorTaskSpec.Package,
		InstanceName:    task.Spec.KudoOperatorTaskSpec.InstanceName,
		AppVersion:      task.Spec.KudoOperatorTaskSpec.AppVersion,
		OperatorVersion: task.Spec.KudoOperatorTaskSpec.OperatorVersion,
		ParameterFile:   task.Spec.KudoOperatorTaskSpec.ParameterFile,
	}, nil
}
