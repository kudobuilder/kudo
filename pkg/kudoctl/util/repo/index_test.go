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

	"github.com/stretchr/testify/assert"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

var update = flag.Bool("update", false, "update .golden files")

func TestParseIndexFile(t *testing.T) {
	indexString := `
apiVersion: v1
entries:
  flink:
  - apiVersion: v1beta1
    name: flink
    urls:
    - https://kudo-repository.storage.googleapis.com/flink-0.1.0.tgz
    operatorVersion: 0.1.0
  - apiVersion: v1beta1
    appVersion: 1.7.2
    name: flink
    urls:
    - https://kudo-repository.storage.googleapis.com/flink-1.7.2_0.1.0.tgz
    operatorVersion: 0.1.0
  kafka:
  - apiVersion: v1beta1
    appVersion: 2.2.1
    name: kafka
    urls:
    - https://kudo-repository.storage.googleapis.com/kafka-2.2.1_0.1.0.tgz
    operatorVersion: 0.1.0
  - apiVersion: v1beta1
    appVersion: 2.2.1
    name: kafka
    urls:
    - https://kudo-repository.storage.googleapis.com/kafka-2.2.1_0.2.0.tgz
    operatorVersion: 0.2.0
  - apiVersion: v1beta1
    appVersion: 2.3.0
    name: kafka
    urls:
    - https://kudo-repository.storage.googleapis.com/kafka-2.3.0_0.2.0.tgz
    operatorVersion: 0.2.0
`
	b := []byte(indexString)
	index, _ := ParseIndexFile(b)

	assert.Equal(t, 2, len(index.Entries), "number of operator entries is 2")

	assert.Equal(t, 2, len(index.Entries["flink"]), "number of flink operators is 2")
	assert.Equal(t, "1.7.2", index.Entries["flink"][0].AppVersion, "flink app version")
	assert.Equal(t, "0.1.0", index.Entries["flink"][0].OperatorVersion, "flink operator version")
	assert.Equal(t, "", index.Entries["flink"][1].AppVersion, "flink app version")
	assert.Equal(t, "0.1.0", index.Entries["flink"][1].OperatorVersion, "flink operator version")

	assert.Equal(t, 3, len(index.Entries["kafka"]), "number of kafka operators is 3")
	assert.Equal(t, "2.3.0", index.Entries["kafka"][0].AppVersion, "kafka app version")
	assert.Equal(t, "0.2.0", index.Entries["kafka"][0].OperatorVersion, "kafka operator version")
	assert.Equal(t, "2.2.1", index.Entries["kafka"][1].AppVersion, "kafka app version")
	assert.Equal(t, "0.2.0", index.Entries["kafka"][1].OperatorVersion, "kafka operator version")
	assert.Equal(t, "2.2.1", index.Entries["kafka"][2].AppVersion, "kafka app version")
	assert.Equal(t, "0.1.0", index.Entries["kafka"][2].OperatorVersion, "kafka operator version")
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

		//nolint:gosec
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
			Name:            name,
			OperatorVersion: version,
			AppVersion:      "0.7.0",
			Description:     "fancy description is here",
			Maintainers: []*kudoapi.Maintainer{
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
		{"duplicate version", dup, "operator 'flink' version: 0.7.0_0.3.0 already exists"},
		{"no version", &missing, "operator 'flink' is missing operator version"},
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
		OperatorVersion:   "1.0.0",
		AppVersion:        "2.2.2",
		KUDOVersion:       "0.5.0",
		KubernetesVersion: "1.15",
		Maintainers:       []*kudoapi.Maintainer{{Name: "Ken Sipe"}},
		URL:               "http://kudo.dev/kafka",
	}
	pf := packages.Files{
		Operator: &o,
	}

	pv := ToPackageVersion(&pf, "1234", "http://localhost")

	assert.Equal(t, o.Name, pv.Name)
	assert.Equal(t, o.OperatorVersion, pv.OperatorVersion)
	assert.Equal(t, o.AppVersion, pv.AppVersion)
	assert.Equal(t, "http://localhost/kafka-2.2.2_1.0.0.tgz", pv.URLs[0])
	assert.Equal(t, "1234", pv.Digest)
}
