// +build integration

package test

import (
	"testing"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
)

func TestHarnessRunIntegration(t *testing.T) {
	harness := Harness{
		TestSuite: kudo.TestSuite{
			CRDDir: "../../config/crds/",
			TestDirs: []string{
				"./test_data/",
			},
			StartKUDO:         false,
			StartControlPlane: true,
		},
		T: t,
	}
	harness.Run()
}
