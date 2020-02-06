package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	v1 "k8s.io/api/core/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/kubectl/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kudobuilder/kudo/pkg/engine/health"
)

// ApplyTask will apply a set of given resources to the cluster. See Run method for more details.
type ApplyTask struct {
	Name      string
	Resources []string
}

// Run method for the ApplyTask. Given the task context, it renders the templates using context parameters
// creates runtime objects and enhances them, and applies them using the controller client. Finally,
// resources are checked for health.
func (at ApplyTask) Run(ctx Context) (bool, error) {
	// 1. - Render task templates -
	rendered, err := render(at.Resources, ctx)
	if err != nil {
		return false, fatalExecutionError(err, taskRenderingError, ctx.Meta)
	}

	// 2. - Enhance them with metadata -
	enhanced, err := enhance(rendered, ctx.Meta, ctx.Enhancer)
	if err != nil {
		return false, fatalExecutionError(err, taskEnhancementError, ctx.Meta)
	}

	// 3. - Apply them using the client -
	applied, err := apply(enhanced, ctx.Client)
	if err != nil {
		return false, err
	}

	// 4. - Check health for all resources -
	err = isHealthy(applied)
	if err != nil {
		// so far we do not distinguish between unhealthy resources and other errors that might occur during a health check
		// an error during a health check is not treated task execution error
		log.Printf("TaskExecution: %v", err)
		return false, nil
	}
	return true, nil
}

func addLastAppliedConfigAnnotation(r runtime.Object) error {
	json, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("failed to marshal obj: %v", err)
	}
	annots, err := metadataAccessor.Annotations(r)
	if err != nil {
		return fmt.Errorf("failed to access annotations: %v", err)
	}
	if annots == nil {
		annots = map[string]string{}
	}

	annots[v1.LastAppliedConfigAnnotation] = string(json)
	if err := metadataAccessor.SetAnnotations(r, annots); err != nil {
		return err
	}

	return nil
}

// apply method takes a slice of k8s object and applies them using passed client. If an object
// doesn't exist it will be created. An already existing object will be patched.
func apply(ro []runtime.Object, c client.Client) ([]runtime.Object, error) {
	applied := make([]runtime.Object, 0)

	for _, r := range ro {
		key, _ := client.ObjectKeyFromObject(r)
		existing := r.DeepCopyObject()

		// if CRD we need to clear then namespace from the copy
		if isClusterResource(r) {
			key.Namespace = ""
		}

		err := c.Get(context.TODO(), key, existing)

		switch {
		case apierrors.IsNotFound(err): // create resource if it doesn't exist
			if err := addLastAppliedConfigAnnotation(r); err != nil {
				return nil, fmt.Errorf("failed to add last applied config annotation: %v", err)
			}

			err = c.Create(context.TODO(), r)
			//// c.Create always overrides the input, in this case, the object that had previously set GVK loses it (at least for integration tests)
			//// and this was causing problems in health module
			//// with error failed to convert *unstructured.Unstructured to *v1.Deployment: Object 'Kind' is missing in 'unstructured object has no kind'
			//// so re-setting the GVK here to be sure
			//// https://github.com/kubernetes/kubernetes/issues/80609
			r.GetObjectKind().SetGroupVersionKind(existing.GetObjectKind().GroupVersionKind())
			if err != nil {
				return nil, err
			}
			applied = append(applied, r)
		case err != nil: // raise any error other than StatusReasonNotFound
			return nil, err
		default: // update existing resource
			//err := patch(r, c)
			err := doStrategicThreewayMergePatch(r, c)
			//err := cmdApply(r, c, scheme, restMapper, config)
			if err != nil {
				return nil, fmt.Errorf("failed to real patch: %v", err)
			}
			applied = append(applied, r)
		}
	}

	return applied, nil
}

func isClusterResource(r runtime.Object) bool {
	// this misses a number of cluster scoped resources
	// this is a temporary fix.  The correct solution will use the DiscoveryInterface
	switch r.(type) {
	case *apiextv1beta1.CustomResourceDefinition:
		return true
	case *v1.Namespace:
		return true
	case *v1.PersistentVolume:
		return true
	}
	return false
}

var metadataAccessor = meta.NewAccessor()

func doStrategicThreewayMergePatch(r runtime.Object, c client.Client) error {
	key, _ := client.ObjectKeyFromObject(r)

	// Fetch current configuration from cluster
	currentObj := r.DeepCopyObject()
	err := c.Get(context.TODO(), key, currentObj)
	if err != nil {
		return fmt.Errorf("failed to get current %v", err)
	}
	current, err := json.Marshal(currentObj)
	if err != nil {
		return fmt.Errorf("failed to marshal current %v", err)
	}

	// Get previous configuration from currentObjs annotation
	original, err := util.GetOriginalConfiguration(currentObj)
	if err != nil {
		return fmt.Errorf("failed to get original configuration %v", err)
	}

	// Get new (modified) configuration
	modified, err := util.GetModifiedConfiguration(r, true, unstructured.UnstructuredJSONScheme)
	if err != nil {
		return fmt.Errorf("failed to get modified config %v", err)
	}

	// Create the three way merge patch
	patchMeta, err := strategicpatch.NewPatchMetaFromStruct(r)
	if err != nil {
		return fmt.Errorf("failed to create patch meta %v", err)
	}

	patchData, err := strategicpatch.CreateThreeWayMergePatch(original, modified, current, patchMeta, true)
	if err != nil {
		// TODO: If we have conflicts when creating the patch, do a delete/create
		return fmt.Errorf("failed to create patch data %v", err)
	}

	//fmt.Printf("Execute Strategic Patch: %s", string(patchData))

	// Execute the patch
	// TODO: If we get an error here, fall back to delete/create?
	return c.Patch(context.TODO(), r, client.ConstantPatch(types.StrategicMergePatchType, patchData))
}

//func isKudoType(object runtime.Object) bool {
//	_, isOperator := object.(*v1beta1.OperatorVersion)
//	_, isOperatorVersion := object.(*v1beta1.Operator)
//	_, isInstance := object.(*v1beta1.Instance)
//	return isOperator || isOperatorVersion || isInstance
//}

func isHealthy(ro []runtime.Object) error {
	for _, r := range ro {
		err := health.IsHealthy(r)
		if err != nil {
			key, _ := client.ObjectKeyFromObject(r)
			return fmt.Errorf("object %s/%s is NOT healthy: %w", key.Namespace, key.Name, err)
		}
	}
	return nil
}
