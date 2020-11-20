package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/engine/resource"
	"github.com/kudobuilder/kudo/pkg/kubernetes/status"
	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

// ApplyTask will apply a set of given resources to the cluster. See Run method for more details.
type ApplyTask struct {
	Name      string
	Resources []string
}

var (
	metadataAccessor = meta.NewAccessor()
)

// Run method for the ApplyTask. Given the task context, it renders the templates using context parameters
// creates runtime objects and enhances them, and applies them using the controller client. Finally,
// resources are checked for health.
func (at ApplyTask) Run(ctx Context) (bool, error) {
	// 1. - Render task templates -
	rendered, err := render(at.Resources, ctx)
	if err != nil {
		return false, fatalExecutionError(err, taskRenderingError, ctx.Meta)
	}

	// 2. - Convert to objects
	objs, err := convert(rendered)
	if err != nil {
		return false, fatalExecutionError(err, taskRenderingError, ctx.Meta)
	}

	// 2. - Enhance them with metadata -
	enhanced, err := enhance(objs, ctx.Meta, ctx.Enhancer)
	if err != nil {
		return false, err
	}

	// 3. - Apply them using the client -
	applied, err := applyResources(enhanced, ctx)
	if err != nil {
		return false, err
	}

	// 4. - Check health for all resources -
	err = isHealthy(applied)
	if err != nil {
		if errors.Is(err, engine.ErrFatalExecution) {
			return false, fatalExecutionError(err, failedTerminalState, ctx.Meta)
		}
		// an error during a health check is not treated task execution error
		log.Printf("TaskExecution: %v", err)
		return false, nil
	}
	return true, nil
}

func addLastAppliedConfigAnnotation(r runtime.Object) error {
	// Serialize object
	rSer, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("failed to marshal obj: %v", err)
	}

	annotations, err := metadataAccessor.Annotations(r)
	if err != nil {
		return fmt.Errorf("failed to access annotations: %v", err)
	}
	if annotations == nil {
		annotations = map[string]string{}
	}

	// Set serialized object as an annotation on itself
	annotations[kudo.LastAppliedConfigAnnotation] = string(rSer)
	if err := metadataAccessor.SetAnnotations(r, annotations); err != nil {
		return err
	}

	return nil
}

// apply method takes a slice of k8s object and applies them using passed client. If an object
// doesn't exist it will be created. An already existing object will be patched.
func applyResources(rr []runtime.Object, ctx Context) ([]runtime.Object, error) {
	applied := make([]runtime.Object, 0)

	for _, r := range rr {
		existing := r.DeepCopyObject()

		key, err := resource.ObjectKeyFromObject(r, ctx.Discovery)
		if err != nil {
			return nil, err
		}

		err = ctx.Client.Get(context.TODO(), key, existing)

		switch {
		case apierrors.IsNotFound(err): // create resource if it doesn't exist
			if err := addLastAppliedConfigAnnotation(r); err != nil {
				return nil, fmt.Errorf("failed to add last applied config annotation to a %s %s: %w", r.GetObjectKind().GroupVersionKind(), key, err)
			}

			err = ctx.Client.Create(context.TODO(), r)
			// c.Create always overrides the input, in this case, the object that had previously set GVK loses it (at least for integration tests)
			// and this was causing problems in health module
			// with error failed to convert *unstructured.Unstructured to *v1.Deployment: Object 'Kind' is missing in 'unstructured object has no kind'
			// so re-setting the GVK here to be sure
			// https://github.com/kubernetes/kubernetes/issues/80609
			r.GetObjectKind().SetGroupVersionKind(existing.GetObjectKind().GroupVersionKind())
			if err != nil {
				return nil, err
			}
			applied = append(applied, r)
		case err != nil: // raise any error other than StatusReasonNotFound
			return nil, err
		default: // update existing resource
			err := patchResource(r, existing, ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to patch a %s %s: %w", r.GetObjectKind().GroupVersionKind(), key, err)
			}
			applied = append(applied, r)
		}
	}

	return applied, nil
}

func patchResource(modifiedObj, currentObj runtime.Object, ctx Context) error {

	// Serialize current configuration
	current, err := json.Marshal(currentObj)
	if err != nil {
		return fatalExecutionError(fmt.Errorf("failed to marshal current %v", err), taskRenderingError, ctx.Meta)
	}

	// Get previous configuration from currentObjs annotation
	original, err := getOriginalConfiguration(currentObj)
	if err != nil {
		return fatalExecutionError(fmt.Errorf("failed to get original configuration %v", err), taskRenderingError, ctx.Meta)
	}

	// Get new (modified) configuration
	modified, err := getModifiedConfiguration(modifiedObj, true, unstructured.UnstructuredJSONScheme)
	if err != nil {
		return fatalExecutionError(fmt.Errorf("failed to get modified config %v", err), taskRenderingError, ctx.Meta)
	}

	// Create the actual patchResource
	var patchData []byte
	var patchType types.PatchType
	if useSimpleThreeWayMerge(modifiedObj) {
		patchType = types.MergePatchType
		patchData, err = simpleThreeWayMergePatch(original, modified, current)
	} else {
		patchType = types.StrategicMergePatchType
		patchData, err = strategicThreeWayMergePatch(modifiedObj, original, modified, current)
	}

	if err != nil {
		// TODO: We could try to delete/create here, but that would be different behavior from before
		return fatalExecutionError(fmt.Errorf("failed to create patch: %v", err), taskRenderingError, ctx.Meta)
	}

	// Execute the patchResource
	err = ctx.Client.Patch(context.TODO(), modifiedObj, client.RawPatch(patchType, patchData))
	if err != nil {
		return fmt.Errorf("failed to execute patch: %v", err)
	}
	return nil
}

func useSimpleThreeWayMerge(newObj runtime.Object) bool {
	_, isUnstructured := newObj.(runtime.Unstructured)
	_, isCRDV1Beta1 := newObj.(*apiextv1beta1.CustomResourceDefinition)
	_, isCRDV1 := newObj.(*apiextv1.CustomResourceDefinition)
	return isUnstructured || isCRDV1Beta1 || isCRDV1 || isKudoType(newObj)
}

func isKudoType(object runtime.Object) bool {
	_, isOperator := object.(*kudoapi.OperatorVersion)
	_, isOperatorVersion := object.(*kudoapi.Operator)
	_, isInstance := object.(*kudoapi.Instance)
	return isOperator || isOperatorVersion || isInstance
}

func simpleThreeWayMergePatch(original, modified, current []byte) ([]byte, error) {
	preconditions := []mergepatch.PreconditionFunc{
		mergepatch.RequireKeyUnchanged("apiVersion"),
		mergepatch.RequireKeyUnchanged("kind"),
		mergepatch.RequireMetadataKeyUnchanged("name"),
	}

	patchData, err := jsonmergepatch.CreateThreeWayJSONMergePatch(original, modified, current, preconditions...)
	if err != nil {
		if mergepatch.IsPreconditionFailed(err) {
			return nil, fmt.Errorf("%s", "At least one of apiVersion, kind and name was changed")
		}
		return nil, fmt.Errorf(" failed to create json merge patch: %v", err)
	}

	return patchData, nil
}

func strategicThreeWayMergePatch(r runtime.Object, original, modified, current []byte) ([]byte, error) {
	// Create the patchResource
	patchMeta, err := strategicpatch.NewPatchMetaFromStruct(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create patch meta %v", err)
	}

	patchData, err := strategicpatch.CreateThreeWayMergePatch(original, modified, current, patchMeta, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create patch data %v", err)
	}

	return patchData, nil
}

func isHealthy(ro []runtime.Object) error {
	for _, r := range ro {
		err := isResourceHealthy(r)
		if err != nil {
			key, _ := client.ObjectKeyFromObject(r) // err not possible as all runtime.Objects have metadata
			return fmt.Errorf("object %s/%s is NOT healthy: %w", key.Namespace, key.Name, err)
		}
	}
	return nil
}

func isResourceHealthy(obj runtime.Object) error {
	healthy, msg, err := status.IsHealthy(obj)
	if err != nil {
		return err
	}
	if healthy {
		if msg != "" {
			log.Printf("HealthUtil: %s", msg)
		}
		return nil
	}
	isTerminal, terminalMsg, err := status.IsTerminallyFailed(obj)
	if err != nil {
		return err
	}
	if isTerminal {
		log.Printf("HealthUtil: %s", terminalMsg)
		return fmt.Errorf("%wHealthUtil: %s", engine.ErrFatalExecution, terminalMsg)
	}

	log.Printf("HealthUtil: %s", msg)
	return errors.New(msg)
}

// copy from k8s.io/kubectl@v0.16.6/pkg/util/apply.go, with adjustments
// GetOriginalConfiguration retrieves the original configuration of the object
// from the annotation.
// returns an error if the annotation is nil or invalid JSON
func getOriginalConfiguration(obj runtime.Object) ([]byte, error) {
	annots, err := metadataAccessor.Annotations(obj)
	if err != nil {
		return nil, err
	}

	if annots == nil {
		return nil, nil
	}

	original, ok := annots[kudo.LastAppliedConfigAnnotation]
	if !ok {
		return nil, fmt.Errorf("LastAppliedConfigAnnotation is not set")
	}

	originalBytes := []byte(original)

	originalMap := map[string]interface{}{}
	if len(original) > 0 {
		if err := json.Unmarshal(originalBytes, &originalMap); err != nil {
			return nil, fmt.Errorf("LastAppliedConfigAnnotation is invalid JSON")
		}
	}

	return []byte(original), nil
}

// copy from k8s.io/kubectl@v0.16.6/pkg/util/apply.go, but with different annotation
// GetModifiedConfiguration retrieves the modified configuration of the object.
// If annotate is true, it embeds the result as an annotation in the modified
// configuration. If an object was read from the command input, it will use that
// version of the object. Otherwise, it will use the version from the server.
func getModifiedConfiguration(obj runtime.Object, annotate bool, codec runtime.Encoder) ([]byte, error) {
	// First serialize the object without the annotation to prevent recursion,
	// then add that serialization to it as the annotation and serialize it again.
	var modified []byte

	// Otherwise, use the server side version of the object.
	// Get the current annotations from the object.
	annots, err := metadataAccessor.Annotations(obj)
	if err != nil {
		return nil, err
	}

	if annots == nil {
		annots = map[string]string{}
	}

	original := annots[kudo.LastAppliedConfigAnnotation]
	delete(annots, kudo.LastAppliedConfigAnnotation)
	if err := metadataAccessor.SetAnnotations(obj, annots); err != nil {
		return nil, err
	}

	modified, err = runtime.Encode(codec, obj)
	if err != nil {
		return nil, err
	}

	if annotate {
		annots[kudo.LastAppliedConfigAnnotation] = string(modified)
		if err := metadataAccessor.SetAnnotations(obj, annots); err != nil {
			return nil, err
		}

		modified, err = runtime.Encode(codec, obj)
		if err != nil {
			return nil, err
		}
	}

	// Restore the object to its original condition.
	annots[kudo.LastAppliedConfigAnnotation] = original
	if err := metadataAccessor.SetAnnotations(obj, annots); err != nil {
		return nil, err
	}

	return modified, nil
}
