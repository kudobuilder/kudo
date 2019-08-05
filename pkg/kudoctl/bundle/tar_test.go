package bundle

import (
	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
	"github.com/spf13/afero"
	"os"
	"testing"
)

func TestRegularFileTarball(t *testing.T) {
	var fs= afero.NewOsFs()
	//files.CopyOperatorToFs(fs, "testdata/zk", "/opt")

	f, _ := fs.Create("/opt/zk.tar.gz")

	// sum of zk.tar.gz (host)
	o, _ := os.Open("testdata/zk.tar.gz")
	expected, _ :=  files.Sha256Sum(o)

	// path is that copied into in-mem fs
	_ = tarballWriter(fs, "testdata/zk", f)
	f.Close()

	f, _ = fs.Open("/opt/zk.tar.gz")
	actual, _ := files.Sha256Sum(f)
	if expected != actual {
		t.Errorf("Expecting the tarball to have same hash as testdata/zk.tar.gz but they differ: %v, %v", expected, actual)
	}
}