package bundle

import (
	"os"
	"testing"

	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
	"github.com/spf13/afero"
)

func TestRegularFileTarball(t *testing.T) {
	var fs = afero.NewMemMapFs()
	files.CopyOperatorToFs(fs, "testdata/zk", "/opt")

	f, _ := fs.Create("/opt/zk.tgz")

	o, _ := os.Open("/opt/zk/operator.yaml")
	expected, _ := files.Sha256Sum(o)

	// path is that copied into in-mem fs
	_ = tarballWriter(fs, "/opt/zk", f)
	f.Close()

	//open for reading in an untar
	f, _ = fs.Open("/opt/zk.tgz")
	defer f.Close()

	Untar(fs, "/opt/untar", f)

	u, _ := os.Open("/opt/untar/operator.yaml")
	actual, _ := files.Sha256Sum(u)

	if expected != actual {
		t.Errorf("Expecting the tarball and untar of operator.yaml to have same hash but they differ: %v, %v", expected, actual)
	}
}
