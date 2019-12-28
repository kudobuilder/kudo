package v1beta1

import (
	"context"
	"fmt"
	"net/http"
	"reflect"

	"k8s.io/api/admission/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +k8s:deepcopy-gen=false

// InstanceValidator holds the data for the instance validator hook
type InstanceValidator struct {
	client  client.Client
	decoder *admission.Decoder
}

// InstanceValidator validates updates to an Instance, guarding from conflicting plan executions
func (v *InstanceValidator) Handle(ctx context.Context, req admission.Request) admission.Response {

	switch req.Operation {
	// we only validate Instance Updates
	case v1beta1.Update:
		old, new := &Instance{}, &Instance{}

		// req.Object contains the updated object
		if err := v.decoder.DecodeRaw(req.Object, new); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		// req.OldObject is populated for DELETE and UPDATE requests
		if err := v.decoder.DecodeRaw(req.OldObject, old); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}

		if err := validateUpdate(old, new); err != nil {
			return admission.Denied(err.Error())
		}
		return admission.Allowed("")
	default:
		return admission.Allowed("")
	}
}

func validateUpdate(old, new *Instance) error {
	// Disallow spec updates when a plan is in progress
	if old.Status.AggregatedStatus.Status.IsRunning() && specChanged(old.Spec, new.Spec) {
		return fmt.Errorf("cannot update Instance %s/%s right now, there's plan %s in progress", old.Namespace, old.Name, old.Status.AggregatedStatus.ActivePlanName)
	}

	return nil
}

func specChanged(old InstanceSpec, new InstanceSpec) bool {
	return !reflect.DeepEqual(old, new)
}

// InstanceValidator implements inject.Client.
// A client will be automatically injected.

// InjectClient injects the client.
func (v *InstanceValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// InstanceValidator implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (v *InstanceValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
