package generate

import (
	"flag"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
)

var updateGolden = flag.Bool("update", false, "update .golden files and manifests in /config/crd")

func TestAddMaintainer(t *testing.T) {
	goldenFile := "maintainer"
	fs := afero.NewMemMapFs()
	files.CopyOperatorToFs(fs, "../../packages/testdata/zk", "/opt")
	m := kudoapi.Maintainer{
		Name:  "Cat in the hat",
		Email: "c@hat.com",
	}

	err := AddMaintainer(fs, "/opt/zk", &m)
	assert.NoError(t, err)

	operator, err := afero.ReadFile(fs, "/opt/zk/operator.yaml")
	assert.NoError(t, err)

	gp := filepath.Join("testdata", goldenFile+".golden")

	if *updateGolden {
		t.Logf("updating golden file %s", goldenFile)

		//nolint:gosec
		if err := ioutil.WriteFile(gp, operator, 0644); err != nil {
			t.Fatalf("failed to update golden file: %s", err)
		}
	}
	g, err := ioutil.ReadFile(gp)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}

	assert.Equal(t, g, operator, "for golden file: %s", gp)
}

func TestListMaintainers(t *testing.T) {
	fs := afero.OsFs{}
	m, err := MaintainerList(fs, "../../packages/testdata/zk")
	assert.NoError(t, err)

	assert.Equal(t, 3, len(m))
	assert.Equal(t, "Alena Varkockova", m[0].Name)
}
