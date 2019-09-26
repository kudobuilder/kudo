package engine

import (
	"errors"
	"fmt"
)

// ApplyTask is a task that attempts to create a set of Kubernetes Resources using a given client
// +k8s:deepcopy-gen=true
type ApplyTask struct {
	Resources []string
	Templates map[string]string
}

// ApplyTask Run
func (c *ApplyTask) Run(ctx Context) error {
	tt := make([]string, len(c.Resources))
	for _, tpl := range c.Resources {
		if t, ok := c.Templates[tpl]; ok {
			tt = append(tt, t)
			continue
		}
		return errors.New(fmt.Sprintf("ApplyTask: Invalid resource reference: %s", tpl))
	}

	pipeline := MultiTask{
		InitialInput: tt,
		Tasks: []TaskBuilder{
			TemplateTaskBuilder,
			RenderTaskBuilder,
		},
	}

	err := pipeline.Run(ctx)
	fmt.Println(err)

	fmt.Println(pipeline.Output())

	return nil

	//pipeline.Run(ctx)
	//
	//kubernetesTask := &KubernetesTask{
	//	Op: "apply",
	//}
	//
	//return kubernetesTask.Run(ctx)

	// run Renderable task (resolves and templates a bunch resources)
	// run Kubernetes task (actually performs the Kubernetes op)
}
