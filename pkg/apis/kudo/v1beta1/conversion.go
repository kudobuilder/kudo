package v1beta1

import (
	"encoding/json"
	"fmt"
	"log"

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

//nolint:stylecheck
func Convert_v1beta1_InstanceSpec_To_kudo_InstanceSpec(in *InstanceSpec, out *kudo.InstanceSpec, s conversion.Scope) error {
	log.Printf("Convert v1beta1.Instance to kudo.Instance")
	if err := autoConvert_v1beta1_InstanceSpec_To_kudo_InstanceSpec(in, out, s); err != nil {
		return err
	}

	params := map[string]interface{}{}
	for k, v := range in.Parameters {
		params[k] = v
	}

	bytes, err := json.Marshal(params)
	if err != nil {
		log.Printf("failed to marshal instance stuff: %v", err)
		return err
	}

	out.Parameters.Raw = bytes

	return nil
}

//nolint:stylecheck
func Convert_v1beta1_OperatorVersionSpec_To_kudo_OperatorVersionSpec(in *OperatorVersionSpec, out *kudo.OperatorVersionSpec, s conversion.Scope) error {
	log.Printf("Convert v1beta1.OperatorVersion to kudo.OperatorVersion")
	if err := autoConvert_v1beta1_OperatorVersionSpec_To_kudo_OperatorVersionSpec(in, out, s); err != nil {
		return err
	}

	schema := map[string]interface{}{}
	schema["title"] = "Parameters"
	schema["description"] = "All Parameters"

	properties := map[string]interface{}{}

	for _, p := range in.Parameters {
		param := map[string]string{}
		param["title"] = p.DisplayName
		if p.DisplayName == "" {
			param["title"] = p.Name
		}
		param["description"] = p.Description
		if p.Default != nil {
			param["default"] = *p.Default
		}
		param["type"] = string(p.Type)
		param["trigger"] = p.Trigger
		if p.Immutable != nil && *p.Immutable {
			param["immutable"] = "true"
		}
		properties[p.Name] = param
	}

	schema["properties"] = properties

	bytes, err := json.Marshal(schema)
	if err != nil {
		log.Printf("failed to marshal ov stuff: %v", err)
		return err
	}

	out.Parameters.Raw = bytes

	return nil
}

//nolint:stylecheck
func Convert_kudo_InstanceSpec_To_v1beta1_InstanceSpec(in *kudo.InstanceSpec, out *InstanceSpec, s conversion.Scope) error {
	return fmt.Errorf("can't convert Hub version to v1beta1")
}

//nolint:stylecheck
func Convert_kudo_OperatorVersionSpec_To_v1beta1_OperatorVersionSpec(int *kudo.OperatorVersionSpec, out *OperatorVersionSpec, s conversion.Scope) error {
	return fmt.Errorf("can't convert Hub version to v1beta1")
}
