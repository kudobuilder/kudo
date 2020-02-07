package renderer

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kudobuilder/kudo/pkg/engine"
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
