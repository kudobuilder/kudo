package verifier

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

type Warning string
type Warnings []Warning
type Error string
type Errors []Error

// PackageVerifier defines the interface for all parameter verifiers
type PackageVerifier interface {
	Verify(pf *packages.Files) (Warnings, Errors)
}

func CreateParamError(param v1beta1.Parameter, reason string) Error {
	return Error(fmt.Sprintf("parameter %q %s", param.Name, reason))
}

func CreateParamWarning(param v1beta1.Parameter, reason string) Warning {
	return Warning(fmt.Sprintf("parameter %q %s", param.Name, reason))
}
