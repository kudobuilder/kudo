package task

import (
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"k8s.io/apimachinery/pkg/runtime"
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
	if err := pipeline.Run(ctx); err != nil {
		return err
	}

	res := pipeline.Output().([]runtime.Object)

	var g errgroup.Group
	var okObjs []runtime.Object
	for _, o := range res {
		o := o
		g.Go(func() error {
			if err := ctx.KubernetesClient.Create(ctx, o); err != nil {
				return err
			}
			okObjs = append(okObjs, o)
			return nil
		})
	}

	if err := g.Wait(); err == nil {
		return nil
	}

	// If we get to here, we have an error creating a resource and we need to delete everything that was created
	// successfully. We'll also need to handle composing a useful error since we can have multiple resources that
	// fail, and if any delete fails, we need to give users the best possible idea why the world is burning down.

	for _, o := range okObjs {
		o := o
		g.Go(func() error {
			if err := ctx.KubernetesClient.Delete(ctx, o); err != nil {
				return err
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}
