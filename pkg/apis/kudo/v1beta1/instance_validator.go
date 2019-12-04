package v1beta1

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/util/kudo"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ kudo.Validator = &Instance{}

// ValidateCreate implements webhookutil.validator (from controller-runtime)
// we do not enforce any rules upon creation right now
func (i *Instance) ValidateCreate(req admission.Request) error {
	return nil
}

// ValidateUpdate hook called when UPDATE operation is triggered and our admission webhook is triggered
// ValidateUpdate implements webhookutil.validator (from controller-runtime)
func (i *Instance) ValidateUpdate(old runtime.Object, req admission.Request) error {
	if i.Status.AggregatedStatus.Status.IsRunning() && req.RequestSubResource != "status" {
		// when updating anything else than status, there shouldn't be a running plan
		return fmt.Errorf("cannot update Instance %s/%s right now, there's plan %s in progress", i.Namespace, i.Name, i.Status.AggregatedStatus.ActivePlanName)
	}
	return nil
}

// ValidateDelete hook called when DELETE operation is triggered and our admission webhook is triggered
// we don't enforce any validation on DELETE right now
// ValidateDelete implements webhookutil.validator (from controller-runtime)
func (i *Instance) ValidateDelete(req admission.Request) error {
	return nil
}
