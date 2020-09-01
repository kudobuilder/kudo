package v1beta2

import (
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	controllerconversion "sigs.k8s.io/controller-runtime/pkg/conversion"
)

var myScheme *runtime.Scheme

func addConversionFuncs(scheme *runtime.Scheme) error {
	myScheme = scheme
	return nil
}

func (i *Instance) ConvertTo(dst controllerconversion.Hub) error {
	return myScheme.Converter().Convert(i, dst, conversion.SourceToDest, nil)
}

func (i *Instance) ConvertFrom(src controllerconversion.Hub) error {
	return myScheme.Converter().Convert(src, i, conversion.DestFromSource, nil)
}

func (o *Operator) ConvertTo(dst controllerconversion.Hub) error {
	return myScheme.Converter().Convert(o, dst, conversion.SourceToDest, nil)
}

func (o *Operator) ConvertFrom(src controllerconversion.Hub) error {
	return myScheme.Converter().Convert(src, o, conversion.DestFromSource, nil)
}

func (ov *OperatorVersion) ConvertTo(dst controllerconversion.Hub) error {
	return myScheme.Converter().Convert(ov, dst, conversion.SourceToDest, nil)
}

func (ov *OperatorVersion) ConvertFrom(src controllerconversion.Hub) error {
	return myScheme.Converter().Convert(src, ov, conversion.DestFromSource, nil)
}
