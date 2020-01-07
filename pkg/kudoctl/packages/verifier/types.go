package verifier

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

// PackageVerifier defines the interface for all parameter verifiers
type PackageVerifier interface {
	Verify(pf *packages.Files) Result
}
