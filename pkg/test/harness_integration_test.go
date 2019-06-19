// +build integration

package test

import (
	"testing"
)

func TestRunWithKudo(t *testing.T) {
	harness := Harness{
		T:                 t,
		CRDDir:            "../../config/crds/",
		ManifestsDir:      "../../config/samples/test-framework/",
		TestDirs:          []string{"./test_data/"},
		StartControlPlane: true,
	}

	harness.Run()
}
