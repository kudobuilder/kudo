package engine

import (
	"bytes"
	"fmt"
	"text/template"

	kudov1alpha1 "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/controller"
	"github.com/masterminds/sprig"
)

// Engine is the control struct for parsing and templating Kubernetes resources in an ordered fashion
type Engine struct {
	FuncMap template.FuncMap
}

// New creates an engine with a default function map, using a modified Sprig func map. Because these
// templates are rendered by the operator, we delete any functions that potentially access the environment
// the controller is running in.
func New() *Engine {
	f := sprig.TxtFuncMap()

	// Prevent environment access inside the running KUDO Controller
	funcs := []string{"env", "expandenv", "base", "dir", "clean", "ext", "isAbs"}

	for _, fun := range funcs {
		delete(f, fun)
	}

	return &Engine{
		FuncMap: f,
	}
}

// Render creates a fully rendered template based on a set of values. It parses these in strict mode,
// returning errors when keys are missing.
func (e *Engine) Render(tpl string, vals map[string]interface{}) (string, error) {
	t := template.New("gotpl")
	t.Option("missingkey=error")

	var buf bytes.Buffer
	t = t.New("tpl").Funcs(e.FuncMap)

	if _, err := t.Parse(tpl); err != nil {
		return "", fmt.Errorf("error parsing template: %s", err)
	}

	if err := t.ExecuteTemplate(&buf, "tpl", vals); err != nil {
		return "", fmt.Errorf("error rendering template: %s", err)
	}

	return buf.String(), nil
}

func ParseConfig(instance *kudov1alpha1.Instance, frameworkVersion *kudov1alpha1.FrameworkVersion, recorder func(eventtype, reason, message string)) (map[string]interface{}, error) {

	//Load parameters:
	//Create config map to hold all parameters for instantiation
	configs := make(map[string]interface{})
	//Default parameters from instance metadata
	configs[controller.FrameworkName] = frameworkVersion.Spec.Framework.Name
	configs[controller.Name] = instance.Name
	configs[controller.Namespace] = instance.Namespace

	params := make(map[string]interface{})
	//parameters from instance spec
	for k, v := range instance.Spec.Parameters {
		if _, ok := configs[k]; ok {
			return nil, fmt.Errorf("cannot overwrite predefined config param %v with new value %v", k, v)
		}
		params[k] = v
	}
	//merge defaults with customizations
	for _, param := range frameworkVersion.Spec.Parameters {
		_, ok := params[param.Name]
		if !ok { //not specified in params
			if param.Required {
				err := fmt.Errorf("parameter %v was required but not provided by instance %v", param.Name, instance.Name)
				recorder(controller.Warning,"MissingParameter",	err.Error())
				return nil, err
			}
			params[param.Name] = param.Default
		}
	}
	configs[controller.Params] = params
	return configs, nil
}
