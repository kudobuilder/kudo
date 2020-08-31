package v1beta1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"

	controllerconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/kudobuilder/kudo/pkg/apis/kudo"
)

var myScheme *runtime.Scheme

func addConversionFuncs(scheme *runtime.Scheme) error {
	myScheme = scheme
	return nil
}

func (i *Instance) ConvertTo(dst controllerconversion.Hub) error {
	return myScheme.Converter().Convert(i, dst, 0, nil)
}

func (i *Instance) ConvertFrom(src controllerconversion.Hub) error {
	return myScheme.Converter().Convert(src, i, 0, nil)
}

func (o *Operator) ConvertTo(dst controllerconversion.Hub) error {
	return myScheme.Converter().Convert(o, dst, 0, nil)
}

func (o *Operator) ConvertFrom(src controllerconversion.Hub) error {
	return myScheme.Converter().Convert(src, o, 0, nil)
}

func (ov *OperatorVersion) ConvertTo(dst controllerconversion.Hub) error {
	return myScheme.Converter().Convert(ov, dst, 0, nil)
}

func (ov *OperatorVersion) ConvertFrom(src controllerconversion.Hub) error {
	return myScheme.Converter().Convert(src, ov, 0, nil)
}

//nolint:golint,stylecheck
func Convert_v1beta1_InstanceSpec_To_kudo_InstanceSpec(in *InstanceSpec, out *kudo.InstanceSpec, s conversion.Scope) error {
	if err := autoConvert_v1beta1_InstanceSpec_To_kudo_InstanceSpec(in, out, s); err != nil {
		return err
	}

	// TODO: Convert Params

	return nil
}

//nolint:golint,stylecheck
func Convert_v1beta1_OperatorVersionSpec_To_kudo_OperatorVersionSpec(in *OperatorVersionSpec, out *kudo.OperatorVersionSpec, s conversion.Scope) error {
	if err := autoConvert_v1beta1_OperatorVersionSpec_To_kudo_OperatorVersionSpec(in, out, s); err != nil {
		return err
	}

	// TODO: Convert Params

	return nil
}

//nolint:golint,stylecheck
func Convert_kudo_InstanceSpec_To_v1beta1_InstanceSpec(in *kudo.InstanceSpec, out *InstanceSpec, s conversion.Scope) error {
	return fmt.Errorf("can't convert Hub version to v1beta1")
}

//nolint:golint,stylecheck
func Convert_kudo_OperatorVersionSpec_To_v1beta1_OperatorVersionSpec(int *kudo.OperatorVersionSpec, out *OperatorVersionSpec, s conversion.Scope) error {
	return fmt.Errorf("can't convert Hub version to v1beta1")
}
