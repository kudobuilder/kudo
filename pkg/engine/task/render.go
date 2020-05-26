package task

import (
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/engine/renderer"
)

// render method takes resource names and Instance parameters and then renders passed templates using kudo engine.
func render(resourceNames []string, ctx Context) (map[string]string, error) {

	configs := renderer.NewVariableMap().
		WithMetadata(ctx.Meta).
		WithParameters(ctx.Parameters).
		WithPipes(ctx.Pipes)

	resources := map[string]string{}
	engine := renderer.New()

	for _, rn := range resourceNames {
		resource, ok := ctx.Templates[rn]

		if !ok {
			return nil, fmt.Errorf("error finding resource named %s", rn)
		}

		rendered, err := engine.Render(rn, resource, configs)
		if err != nil {
			return nil, fmt.Errorf("error expanding template %s: %w", rn, err)
		}

		resources[rn] = rendered
	}
	return resources, nil
}

// enhance method takes a slice of rendered templates, applies conventions using Enhancer and
// returns a slice of k8s objects.
func enhance(rendered map[string]string, meta renderer.Metadata, enhancer renderer.Enhancer) ([]runtime.Object, error) {
	enhanced, err := enhancer.Apply(rendered, meta)

	switch {
	case errors.Is(err, engine.ErrFatalExecution):
		return nil, fatalExecutionError(err, taskEnhancementError, meta)
	case err != nil:
		return nil, err
	}

	return enhanced, err
}
