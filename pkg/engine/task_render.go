package engine

import (
	"errors"

	"github.com/kudobuilder/kudo/pkg/util/template"
	"k8s.io/apimachinery/pkg/runtime"
)

type templates []string

// RenderTask is a task that, when given a set of templates, will take parameters from the Context and template them out using KUDO's builtin templating engine.
type RenderTask struct {
	Templates       templates
	renderedObjects []runtime.Object
}

func RenderTaskBuilder(input interface{}) (Tasker, error) {
	if coerced, ok := input.([]string); ok {
		return &RenderTask{Templates: coerced}, nil
	}
	return nil, errors.New("RenderTaskBuilder: could not coerce input to templates (type []string)")
}

func (e *RenderTask) Run(ctx Context) error {
	for _, t := range e.Templates {
		obj, err := template.ParseKubernetesObjects(t)
		if err != nil {
			return err
		}

		e.renderedObjects = append(e.renderedObjects, obj...)
	}
	return nil
}
