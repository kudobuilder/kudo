package generate

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
)

func TestAddPlan(t *testing.T) {
	goldenFile := "plan"
	path := "/opt/zk"
	fs := afero.NewMemMapFs()
	files.CopyOperatorToFs(fs, "../../packages/testdata/zk", "/opt")

	name := "flush"
	p := planToFlush()

	err := AddPlan(fs, path, name, &p)
	assert.NoError(t, err)

	params, err := afero.ReadFile(fs, "/opt/zk/operator.yaml")
	assert.NoError(t, err)

	gp := filepath.Join("testdata", goldenFile+".golden")

	if *updateGolden {
		t.Logf("updating golden file %s", goldenFile)

		//nolint:gosec
		if err := ioutil.WriteFile(gp, params, 0644); err != nil {
			t.Fatalf("failed to update golden file: %s", err)
		}
	}
	golden, err := ioutil.ReadFile(gp)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}

	assert.Equal(t, golden, params, "for golden file: %s", gp)
}

func planToFlush() kudoapi.Plan {
	steps := []kudoapi.Step{{
		Name:  "push-lever",
		Tasks: []string{"lower-lever"},
	}}

	phases := []kudoapi.Phase{{
		Name:     "flush",
		Strategy: "serial",
		Steps:    steps,
	}}

	p := kudoapi.Plan{
		Strategy: "serial",
		Phases:   phases,
	}
	return p
}

func TestAddPlan_bad_path(t *testing.T) {
	path := "."
	fs := afero.OsFs{}

	name := "flush"
	p := planToFlush()

	err := AddPlan(fs, path, name, &p)
	assert.Error(t, err)
}

func TestListPlanNames(t *testing.T) {
	fs := afero.OsFs{}
	p, err := PlanNameList(fs, "../../packages/testdata/zk")
	assert.NoError(t, err)

	assert.Equal(t, 2, len(p))
	assert.Equal(t, "deploy", p[0])
}

func TestListPlans(t *testing.T) {
	fs := afero.OsFs{}
	planMap, err := PlanList(fs, "../../packages/testdata/zk")
	assert.NoError(t, err)

	assert.Equal(t, 2, len(planMap))
	assert.Equal(t, "zookeeper", planMap["deploy"].Phases[0].Name)
}
