package engine

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/masterminds/sprig"
)

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
		task.delete(f, fun)
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
