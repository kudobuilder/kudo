package v1beta1

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

// this forces the instance type to implement Validator interface, we'll get compile time error if it's not true anymore
var _ kudo.Validator = &Instance{}

// ValidateCreate implements kudo.Validator (slightly tweaked interface originally from controller-runtime)
// we do not enforce any rules upon creation right now
func (in *Instance) ValidateCreate(req admission.Request) error {
	return nil
}

// ValidateUpdate hook called when UPDATE operation is triggered and our admission webhook is triggered
// ValidateUpdate implements kudo.Validator (slightly tweaked interface originally from controller-runtime)
func (in *Instance) ValidateUpdate(old runtime.Object, req admission.Request) error {
	oldInstance := old.(*Instance)
	if in.Status.AggregatedStatus.Status.IsRunning() && specChanged(in.Spec, oldInstance.Spec) {
		// when updating anything else than status, there shouldn't be a running plan
		return fmt.Errorf("cannot update Instance %s/%s right now, there's plan %s in progress", in.Namespace, in.Name, in.Status.AggregatedStatus.ActivePlanName)
	}
	return nil
}

func specChanged(old InstanceSpec, new InstanceSpec) bool {
	return !reflect.DeepEqual(old, new)
}

// ValidateDelete hook called when DELETE operation is triggered and our admission webhook is triggered
// we don't enforce any validation on DELETE right now
// ValidateDelete implements kudo.Validator (slightly tweaked interface originally from controller-runtime)
func (in *Instance) ValidateDelete(req admission.Request) error {
	return nil
}
