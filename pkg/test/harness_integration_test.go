// +build integration

package test

import (
	"testing"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
)

func TestRunWithKudo(t *testing.T) {
	harness := Harness{
		T: t,
		TestSuite: kudo.TestSuite{
			CRDDir:            "../../config/crds/",
			ManifestsDir:      "../../config/samples/test-framework/",
			TestDirs:          []string{"./test_data/"},
			StartControlPlane: true,
		},
	}

	harness.Run()
}
