package engine

import "errors"

// TemplateTask is a task that, when given a set of templates, will take parameters from the Context and template them out using KUDO's builtin templating engine.
type TemplateTask struct {
	Templates         templates
	renderedTemplates []string
}

func TemplateTaskBuilder(input interface{}) (Tasker, error) {
	if coerced, ok := input.(templates); ok {
		return &TemplateTask{Templates: coerced}, nil
	}
	return nil, errors.New("TemplateTaskBuilder: could not coerce input to templates (type []string)")
}

func (e *TemplateTask) Run(ctx Context) error {
	engine := New()
	for _, t := range e.Templates {
		tpl, err := engine.Render(t, map[string]interface{}{})
		if err != nil {
			return err
		}
		e.renderedTemplates = append(e.renderedTemplates, tpl)
	}
	return nil
}

func (e *TemplateTask) Output() interface{} {
	return e.renderedTemplates
}
