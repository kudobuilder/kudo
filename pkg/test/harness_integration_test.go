// +build integration

package test

import (
	"testing"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
)

func TestHarnessRunIntegration(t *testing.T) {
	harness := Harness{
		TestSuite: kudo.TestSuite{
			CRDDir: "../../config/crds/",
			TestDirs: []string{
				"./test_data/",
			},
			StartKUDO:         true,
			StartControlPlane: true,
		},
		T: t,
	}
	harness.Run()
}
