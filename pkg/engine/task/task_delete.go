package task

import (
	"context"
	"fmt"
	"log"

	"github.com/kudobuilder/kudo/pkg/kubernetes/status"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kudobuilder/kudo/pkg/engine/resource"
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

	// 2. - Convert to objs
	objs, err := convert(rendered)
	if err != nil {
		return false, fatalExecutionError(err, taskRenderingError, ctx.Meta)
	}

	// 3. - Filter unknown objects
	objs, err = filterUnknownObjectTypes(objs, ctx)
	if err != nil {
		return false, fatalExecutionError(err, taskRenderingError, ctx.Meta)
	}

	// 4. - Enhance objects, required for namespaces
	objs, err = ctx.Enhancer.Apply(objs, ctx.Meta)
	if err != nil {
		return false, err
	}

	// 5. - Delete them using the client -
	err = deleteResource(objs, ctx.Client)
	if err != nil {
		return false, err
	}

	// 6. - Check health - wait for object deletion
	return allObjsDeleted(ctx, objs)
}

func allObjsDeleted(ctx Context, objs []runtime.Object) (bool, error) {
	for _, obj := range objs {
		objDeleted, _, err := status.IsDeleted(ctx.Client, ctx.Discovery, obj)
		if err != nil {
			log.Printf("TaskExecution: wait for object deletion: %v", err)
			return false, nil
		}
		if !objDeleted {
			return false, nil
		}
	}
	return true, nil
}

func filterUnknownObjectTypes(objs []runtime.Object, ctx Context) ([]runtime.Object, error) {
	knownObjs := make([]runtime.Object, 0)
	for _, o := range objs {
		isKnown, err := resource.IsKnownObjectType(o, ctx.Discovery)
		if err != nil {
			return nil, fmt.Errorf("failed to detect if object type is known for %s: %v", o.GetObjectKind(), err)
		}
		if isKnown {
			knownObjs = append(knownObjs, o)
		}
	}
	return knownObjs, nil
}

func deleteResource(ro []runtime.Object, c client.Client) error {
	for _, r := range ro {
		err := c.Delete(context.TODO(), r, client.PropagationPolicy(metav1.DeletePropagationForeground))
		if !apierrors.IsNotFound(err) && err != nil {
			return err
		}
	}

	return nil
}
