package renderer

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

// Metadata contains Metadata along with specific fields associated with current plan
// being executed like current plan, phase, step or task names.
type Metadata struct {
	engine.Metadata

	PlanName  string
	PlanUID   types.UID
	PhaseName string
	StepName  string
	TaskName  string
}

// Engine is the control struct for parsing and templating Kubernetes resources in an ordered fashion
type Engine struct {
	FuncMap template.FuncMap
}

// VariableMap is the map of variables which are used in templates like map["OperatorName"]
type VariableMap map[string]interface{}

// New creates an engine with a default function map, using a modified Sprig func map. Because these
// templates are rendered by the operator, we delete any functions that potentially access the environment
// the controller is running in.
func New() *Engine {
	f := sprig.TxtFuncMap()

	f["toYaml"] = ToYaml

	// Prevent environment access inside the running KUDO Controller
	funcs := []string{"env", "expandenv", "base", "dir", "clean", "ext", "isAbs"}

	for _, fun := range funcs {
		delete(f, fun)
	}

	return &Engine{
		FuncMap: f,
	}
}

// Template provides access to the engines template engine.
func (e Engine) Template(name string) *template.Template {
	t := template.New("gotpl")
	t.Option("missingkey=error")

	return t.New(name).Funcs(e.FuncMap)
}

// Render creates a fully rendered template based on a set of values. It parses these in strict mode,
// returning errors when keys are missing.
func (e *Engine) Render(tplName string, tpl string, vals map[string]interface{}) (string, error) {
	var buf bytes.Buffer
	t := e.Template(tplName)

	if _, err := t.Parse(tpl); err != nil {
		return "", fmt.Errorf("error parsing template: %s", err)
	}

	if err := t.ExecuteTemplate(&buf, tplName, vals); err != nil {
		return "", fmt.Errorf("error rendering template: %s", err)
	}

	return buf.String(), nil
}

// NewVariableMap creates variable map necessary for template rendering
// it uses a builder pattern to create the desired map of variables
// for a map of default values `renderer.NewVariableMap().WithDefaults()`
// as a builder pattern, when chaining latter methods in the chain take precedence
// `renderer.NewVariableMap().WithDefaults().WithResource(pkg.Resources)` will have default values overwritten with resource data
func NewVariableMap() VariableMap {
	return make(map[string]interface{})
}

// WithDefaults defines variables which are potentially required by any operator.  By defaulting to this map,
// all templates should pass, even if values are not expected.
func (m VariableMap) WithDefaults() VariableMap {
	m.WithInstance("OperatorName", "Name", "Namespace", "AppVersion", "OperatorVersion")
	m["PlanName"] = "PlanName"
	m["PhaseName"] = "PhaseName"
	m["StepName"] = "StepName"
	return m
}

// WithInstance provides a convince for add instance information.
func (m VariableMap) WithInstance(operatorName, instanceName, namespace, appVersion, operatorVersion string) VariableMap {
	m["OperatorName"] = operatorName
	m["Name"] = instanceName
	m["Namespace"] = namespace
	m["AppVersion"] = appVersion
	m["OperatorVersion"] = operatorVersion

	return m

}

// WithMetadata overrides the map with metadata data
func (m VariableMap) WithMetadata(meta Metadata) VariableMap {
	m["OperatorName"] = meta.OperatorName
	m["Name"] = meta.InstanceName
	m["Namespace"] = meta.InstanceNamespace
	m["AppVersion"] = meta.AppVersion
	m["OperatorVersion"] = meta.OperatorVersion
	m["PlanName"] = meta.PlanName
	m["PhaseName"] = meta.PhaseName
	m["StepName"] = meta.StepName
	m["AppVersion"] = meta.AppVersion
	return m
}

// WithResource overrides map with resource data
func (m VariableMap) WithResource(resources *packages.Resources) VariableMap {
	m["OperatorName"] = resources.Operator.Name
	m["AppVersion"] = resources.OperatorVersion.Spec.AppVersion
	m["OperatorVersion"] = resources.OperatorVersion.Spec.Version
	return m
}

// WithParameters overrides the map with parameter map which uses `interface{}` values
func (m VariableMap) WithParameters(parameters map[string]interface{}) VariableMap {
	m["Params"] = parameters
	return m
}

// WithParameterStrings overrides the map with parameter map which uses `string values
func (m VariableMap) WithParameterStrings(parameters map[string]string) VariableMap {
	m["Params"] = parameters
	return m
}

// WithPipes overrides the map with a pipe map
func (m VariableMap) WithPipes(pipes map[string]string) VariableMap {
	m["Pipes"] = pipes
	return m
}
