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

	f, _ := fs.Create("/opt/zk.tar.gz")

	// sum of zk.tar.gz (host)
	o, _ := os.Open("testdata/zk.tar.gz")
	expected, _ := files.Sha256Sum(o)

	// path is that copied into in-mem fs
	_ = tarballWriter(fs, "/opt/zk", f)
	f.Close()

	f, _ = fs.Open("/opt/zk.tar.gz")
	defer  f.Close()
	
	actual, _ := files.Sha256Sum(f)
	if expected != actual {
		t.Errorf("Expecting the tarball to have same hash as testdata/zk.tar.gz but they differ: %v, %v", expected, actual)
	}
}
