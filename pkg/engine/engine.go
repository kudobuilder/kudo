package engine

import (
	"bytes"
	"fmt"
	"github.com/masterminds/sprig"
	"text/template"
)

type Engine struct {
	FuncMap template.FuncMap
}

func New() *Engine {
	f := sprig.TxtFuncMap()

	// Prevent environment access inside the running KUDO Controller
	delete(f, "env")
	delete(f, "expandenv")

	return &Engine{
		FuncMap: f,
	}
}

func (e *Engine) Render(tpl string, vals map[string]interface{}) (string, error) {
	t := template.New("gotpl")
	t.Option("missingkey=error")


	var buf bytes.Buffer
	t = t.New("tpl").Funcs(e.FuncMap)


	if _, err := t.Parse(tpl); err != nil {
		return "", fmt.Errorf("Error parsing template: %s", err)
	}

	if err := t.ExecuteTemplate(&buf, "tpl", vals); err != nil {
		return "", fmt.Errorf("Error rendering template: %s", err)
	}

	return buf.String(), nil
}