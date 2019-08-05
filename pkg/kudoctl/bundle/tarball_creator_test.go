package bundle

import (
	"crypto/sha256"
	"github.com/spf13/afero"
	"io"
	"os"
	"testing"
)

func TestRegularFileTarball(t *testing.T) {
	var appFs = afero.NewMemMapFs()
	file, _ := appFs.Create("target.tar.gz")
	defer file.Close()

	_ = RegularFileTarball("testdata/zk", file)
	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		t.Fatalf("Error while computing hash of file %v", err)
	}

	actual, err := os.Open("testdata/zk.tar.gz")
	if err != nil {
		t.Fatalf("Error while reading testdata/zk.tar.gz %v", err)
	}
	h2 := sha256.New()
	if _, err := io.Copy(actual, file); err != nil {
		t.Fatalf("Error while computing hash of file %v", err)
	}

	// TODO we need to actually convert h and h2 to string and compare that
	// if that does not work, we need to figure out different way how to verify that the tarball is what we expect it to be
	if h != h2 {
		t.Errorf("Expecting the tarball to have same hash as testdata/zk.tar.gz but they differ: %v, %v", h, h2)
	}
}
