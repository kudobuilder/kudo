package verifier

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

var (
	// Implicits is a set of usable implicits defined in render.go
	Implicits = map[string]bool{
		"Name":         true, // instance name
		"Namespace":    true,
		"OperatorName": true,
		"Params":       true,
		"PlanName":     true,
		"PhaseName":    true,
		"StepName":     true,
		"AppVersion":   true,
	}
)

type ParamWarning string
type ParamWarnings []ParamWarning
type ParamError string
type ParamErrors []ParamError

// PackageVerifier defines the interface for all parameter verifiers
type PackageVerifier interface {
	Verify(pf *packages.Files) (ParamWarnings, ParamErrors)
}

func CreateParamError(param v1beta1.Parameter, reason string) ParamError {
	return ParamError(fmt.Sprintf("parameter %q %s", param.Name, reason))
}

func CreateParamWarning(param v1beta1.Parameter, reason string) ParamWarning {
	return ParamWarning(fmt.Sprintf("parameter %q %s", param.Name, reason))
}
