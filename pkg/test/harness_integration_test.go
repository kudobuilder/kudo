// +build integration

package test

import (
	"testing"

	harness "github.com/kudobuilder/kudo/pkg/apis/testharness/v1beta1"
)

func TestHarnessRunIntegration(t *testing.T) {
	harness := Harness{
		TestSuite: harness.TestSuite{
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
