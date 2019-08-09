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

	"github.com/magiconair/properties/assert"
)

var update = flag.Bool("update", false, "update .golden files")

func TestParseIndexFile(t *testing.T) {
	indexString := `
apiVersion: v1
entries:
  flink:
  - apiVersion: v1alpha1
    appVersion: 1.7.2
    name: flink
    urls:
    - https://kudo-repository.storage.googleapis.com/flink-0.1.0.tgz
    version: 0.1.0
  kafka:
  - apiVersion: v1alpha1
    appVersion: 2.2.1
    name: kafka
    urls:
    - https://kudo-repository.storage.googleapis.com/kafka-0.1.0.tgz
    version: 0.1.0
  - apiVersion: v1alpha1
    appVersion: 2.3.0
    name: kafka
    urls:
    - https://kudo-repository.storage.googleapis.com/kafka-0.2.0.tgz
    version: 0.2.0
`
	b := []byte(indexString)
	index, _ := parseIndexFile(b)

	assert.Equal(t, len(index.Entries), 2, "number of operator entries is 2")
	assert.Equal(t, len(index.Entries["kafka"]), 2, "number of kafka operators is 2")
	assert.Equal(t, index.Entries["flink"][0].AppVersion, "1.7.2", "flink app version")
}

func TestWriteIndexFile(t *testing.T) {
	file := "flink-index.yaml"
	// Given Index with an operator
	index := getTestIndexFile()

	// Setup buffer to marshal yaml to
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	writeIndexFile(&index, w)
	w.Flush()

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

func getTestIndexFile() IndexFile {
	date, _ := time.Parse(time.RFC822, "09 Aug 19 15:04 UTC")

	bv := getTestBundleVersion("flink", "0.3.0")
	bvs := PackageVersions{&bv}
	entries := make(map[string]PackageVersions)
	entries["flink"] = bvs
	index := IndexFile{
		APIVersion: "v1",
		Entries:    entries,
		Generated:  &date,
	}
	return index
}

func getTestBundleVersion(name string, version string) PackageVersion {
	urls := []string{fmt.Sprintf("http://kudo.dev/%v", name)}
	bv := PackageVersion{
		Metadata: &Metadata{
			Name:    name,
			Version: version,
		},
		URLs:       urls,
		APIVersion: "v1aplha1",
		AppVersion: "0.7.0",
	}
	return bv
}

func TestAddBundleVersionErrorConditions(t *testing.T) {
	index := getTestIndexFile()
	dup := index.Entries["flink"][0]
	missing := getTestBundleVersion("flink", "")
	good := getTestBundleVersion("flink", "1.0.0")
	g2 := getTestBundleVersion("kafka", "1.0.0")

	tests := []struct {
		name   string
		bundle *PackageVersion
		err    string
	}{
		{"duplicate version", dup, "operator 'flink' version: 0.3.0 already exists"},
		{"no version", &missing, "operator 'flink' is missing version"},
		{"good additional version", &good, ""},
		{"good additional package", &g2, ""},
	}

	for _, tt := range tests {
		err := index.addBundleVersion(tt.bundle)
		if err != nil && err.Error() != tt.err {
			t.Errorf("%s: expecting error %s got %v", tt.name, tt.err, err)
		}
	}
}
