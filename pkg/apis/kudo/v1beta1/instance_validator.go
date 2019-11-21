package v1beta1

import (
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/kudobuilder/kudo/pkg/util/kudo"

	"k8s.io/apimachinery/pkg/runtime"
)

var _ kudo.Validator = &Instance{}

// ValidateCreate implements webhookutil.validator (from controller-runtime)
// we do not enforce any rules upon creation right now
func (c *Instance) ValidateCreate(req admission.Request) error {
	return nil
}

// ValidateUpdate hook called when UPDATE operation is triggered and our admission webhook is triggered
// ValidateUpdate implements webhookutil.validator (from controller-runtime)
func (c *Instance) ValidateUpdate(old runtime.Object, req admission.Request) error {
	if c.Status.AggregatedStatus.Status.IsRunning() && req.RequestSubResource != "status" {
		// when updating anything else than status, there shouldn't be a running plan
		return fmt.Errorf("cannot update Instance %s/%s right now, there's plan %s in progress", c.Namespace, c.Name, c.Status.AggregatedStatus.ActivePlanName)
	}
	oldInstance := old.(*Instance)
	if c.Name != oldInstance.Name || c.Namespace != oldInstance.Namespace {
		return fmt.Errorf("cannot change the name and/or namespace of Instance. Old instance: %s/%s. New Instance: %s/%s", c.Namespace, c.Name, oldInstance.Namespace, oldInstance.Name)
	}
	return errors.New("always error")
}

// ValidateDelete hook called when DELETE operation is triggered and our admission webhook is triggered
// we don't enforce any validation on DELETE right now
// ValidateDelete implements webhookutil.validator (from controller-runtime)
func (c *Instance) ValidateDelete(req admission.Request) error {
	return nil
}
