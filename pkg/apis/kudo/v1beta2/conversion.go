package v1beta2

import (
	"k8s.io/apimachinery/pkg/runtime"
	controllerconversion "sigs.k8s.io/controller-runtime/pkg/conversion"
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
