package task

import (
	"golang.org/x/net/context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeleteTask will delete a set of given resources from the cluster. See Run method for more details.
type DeleteTask struct {
	Name      string
	Resources []string
}

// Run method for the DeleteTask. Given the task context, it renders the templates using context parameters
// creates runtime objects and enhances them, and finally removes them using the controller client.
func (dt DeleteTask) Run(ctx Context) (bool, error) {
	// 1. - Render task templates -
	rendered, err := render(dt.Resources, ctx)
	if err != nil {
		return false, fatalExecutionError(err, taskRenderingError, ctx.Meta)
	}

	// 2. - Enhance them with metadata -
	enhanced, err := enhance(rendered, ctx.Meta, ctx.Enhancer)
	if err != nil {
		return false, fatalExecutionError(err, taskEnhancementError, ctx.Meta)
	}

	// 3. - Delete them using the client -
	err = delete(enhanced, ctx.Client)
	if err != nil {
		return false, err
	}

	// 4. - Check health: always true for Delete task -
	return true, nil
}

func delete(ro []runtime.Object, c client.Client) error {
	for _, r := range ro {
		err := c.Delete(context.TODO(), r, client.PropagationPolicy(metav1.DeletePropagationForeground))
		if !apierrors.IsNotFound(err) && err != nil {
			return err
		}
	}

	return nil
}
