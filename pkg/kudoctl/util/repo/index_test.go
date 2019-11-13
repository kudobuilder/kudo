package repo

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"

	"github.com/magiconair/properties/assert"
)

var update = flag.Bool("update", false, "update .golden files")

func TestParseIndexFile(t *testing.T) {
	indexString := `
apiVersion: v1
entries:
  flink:
  - apiVersion: v1beta1
    appVersion: 1.7.2
    name: flink
    urls:
    - https://kudo-repository.storage.googleapis.com/flink-0.1.0.tgz
    version: 0.1.0
  kafka:
  - apiVersion: v1beta1
    appVersion: 2.2.1
    name: kafka
    urls:
    - https://kudo-repository.storage.googleapis.com/kafka-0.1.0.tgz
    version: 0.1.0
  - apiVersion: v1beta1
    appVersion: 2.3.0
    name: kafka
    urls:
    - https://kudo-repository.storage.googleapis.com/kafka-0.2.0.tgz
    version: 0.2.0
`
	b := []byte(indexString)
	index, _ := ParseIndexFile(b)

	assert.Equal(t, len(index.Entries), 2, "number of operator entries is 2")
	assert.Equal(t, len(index.Entries["kafka"]), 2, "number of kafka operators is 2")
	assert.Equal(t, index.Entries["flink"][0].AppVersion, "1.7.2", "flink app version")
}

// TestParsingGoldenIndex and parses the index file catching marshalling issues.
func TestParsingGoldenIndex(t *testing.T) {

	file := "flink-index.yaml"

	gp := filepath.Join("testdata", file+".golden")

	g, err := ioutil.ReadFile(gp)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}
	_, err = ParseIndexFile(g)
	if err != nil {
		t.Fatalf("Unable to parse Index file %s", err)
	}
}

func TestWriteIndexFile(t *testing.T) {
	file := "flink-index.yaml"
	// Given Index with an operator
	index, err := getTestIndexFile()
	if err != nil {
		t.Fatal(err)
	}

	// Setup buffer to marshal yaml to
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)

	if err := index.Write(w); err != nil {
		t.Fatal(err)
	}

	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}

	gp := filepath.Join("testdata", file+".golden")

	if *update {
		t.Log("update golden file")
		if err := ioutil.WriteFile(gp, buf.Bytes(), 0644); err != nil {
			t.Fatalf("failed to update golden file: %s", err)
		}
	}
	g, err := ioutil.ReadFile(gp)
	if err != nil {
		t.Fatalf("failed reading .golden: %s", err)
	}
	t.Log(buf.String())
	if !bytes.Equal(buf.Bytes(), g) {
		t.Errorf("json does not match .golden file")
	}
}

func getTestIndexFile() (*IndexFile, error) {
	date, _ := time.Parse(time.RFC822, "09 Aug 19 15:04 UTC")
	index := newIndexFile(&date)
	pv := getTestPackageVersion("flink", "0.3.0")
	if err := index.AddPackageVersion(&pv); err != nil {
		return nil, err
	}
	return index, nil
}

func getTestPackageVersion(name string, version string) PackageVersion {
	urls := []string{fmt.Sprintf("http://kudo.dev/%v", name)}
	bv := PackageVersion{
		Metadata: &Metadata{
			Name:        name,
			Version:     version,
			AppVersion:  "0.7.0",
			Description: "fancy description is here",
			Maintainers: []*v1beta1.Maintainer{
				{Name: "Fabian Baier", Email: "<fabian@mesosphere.io>"},
				{Name: "Tom Runyon", Email: "<runyontr@gmail.com>"},
				{Name: "Ken Sipe", Email: "<kensipe@gmail.com>"}},
		},
		URLs:    urls,
		Removed: false,
		Digest:  "0787a078e64c73064287751b833d63ca3d1d284b4f494ebf670443683d5b96dd",
	}
	return bv
}

func TestAddPackageVersionErrorConditions(t *testing.T) {
	index, err := getTestIndexFile()
	if err != nil {
		t.Fatal(err)
	}
	dup := index.Entries["flink"][0]
	missing := getTestPackageVersion("flink", "")
	good := getTestPackageVersion("flink", "1.0.0")
	g2 := getTestPackageVersion("kafka", "1.0.0")

	tests := []struct {
		name string
		pv   *PackageVersion
		err  string
	}{
		{"duplicate version", dup, "operator 'flink' version: 0.3.0 already exists"},
		{"no version", &missing, "operator 'flink' is missing version"},
		{"good additional version", &good, ""},
		{"good additional package", &g2, ""},
	}

	for _, tt := range tests {
		err := index.AddPackageVersion(tt.pv)
		if err != nil && err.Error() != tt.err {
			t.Errorf("%s: expecting error %s got %v", tt.name, tt.err, err)
		}
	}
}

func TestMapPackageFileToPackageVersion(t *testing.T) {
	o := packages.OperatorFile{
		APIVersion:        packages.APIVersion,
		Name:              "kafka",
		Description:       "",
		Version:           "1.0.0",
		AppVersion:        "2.2.2",
		KUDOVersion:       "0.5.0",
		KubernetesVersion: "1.15",
		Maintainers:       []*v1beta1.Maintainer{{Name: "Ken Sipe"}},
		URL:               "http://kudo.dev/kafka",
	}
	pf := packages.Files{
		Operator: &o,
	}

	pv := ToPackageVersion(&pf, "1234", "http://localhost")

	assert.Equal(t, pv.Name, o.Name)
	assert.Equal(t, pv.Version, o.Version)
	assert.Equal(t, pv.URLs[0], "http://localhost/kafka-1.0.0.tgz")
	assert.Equal(t, pv.Digest, "1234")
}
