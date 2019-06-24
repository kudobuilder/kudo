package repo

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"
)

const (
	frameworkGolden = "framework.golden"
	versionGolden   = "frameworkversion.golden"
	instanceGolden  = "instance.golden"
)

func TestReadFileSystemPackage(t *testing.T) {
	tests := []struct {
		name               string
		v1PackageFolder    string
		outputGoldenFolder string
	}{
		{"zookeeper", "testdata/zk", "testdata/zk-crd-golden"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s-from-filesystem", tt.name), func(t *testing.T) {
			actual, err := ReadFileSystemPackage(tt.v1PackageFolder)
			if err != nil {
				t.Fatalf("Found unexpected error: %v", err)
			}
			golden, err := loadCrdsFromPath(tt.outputGoldenFolder)
			if err != nil {
				t.Fatalf("Found unexpected error when loading golden files: %v", err)
			}
			// we need to sort here because current yaml parsing is not preserving the order of fields
			// at the same time, the deep library we use for equality does not support ignoring order
			sort.Slice(actual.FrameworkVersion.Spec.Parameters, func(i, j int) bool {
				return actual.FrameworkVersion.Spec.Parameters[i].Name < actual.FrameworkVersion.Spec.Parameters[j].Name
			})
			sort.Slice(golden.FrameworkVersion.Spec.Parameters, func(i, j int) bool {
				return golden.FrameworkVersion.Spec.Parameters[i].Name < golden.FrameworkVersion.Spec.Parameters[j].Name
			})

			if diff := deep.Equal(golden, actual); diff != nil {
				t.Error(diff)
			}
		})

		t.Run(fmt.Sprintf("%s-from-tarball", tt.name), func(t *testing.T) {
			appFS := afero.NewMemMapFs()
			file, err := appFS.Create("testtarball.tar.gz")
			if err != nil {
				t.Fatalf("cannot create file for tarball serialization: %+v", err)
			}
			defer file.Close()
			err = createTarball(file, tt.v1PackageFolder)
			if err != nil {
				t.Fatalf("cannot create tarball: %+v", err)
			}
			file, err = appFS.Open("testtarball.tar.gz")
			if err != nil {
				t.Fatalf("could not re-open tarball: %+v", err)
			}
			actual, err := ReadTarGzPackage(file)
			if err != nil {
				t.Fatalf("Found unexpected error: %v", err)
			}
			golden, err := loadCrdsFromPath(tt.outputGoldenFolder)
			if err != nil {
				t.Fatalf("Found unexpected error when loading golden files: %v", err)
			}
			// we need to sort here because current yaml parsing is not preserving the order of fields
			// at the same time, the deep library we use for equality does not support ignoring order
			sort.Slice(actual.FrameworkVersion.Spec.Parameters, func(i, j int) bool {
				return actual.FrameworkVersion.Spec.Parameters[i].Name < actual.FrameworkVersion.Spec.Parameters[j].Name
			})
			sort.Slice(golden.FrameworkVersion.Spec.Parameters, func(i, j int) bool {
				return golden.FrameworkVersion.Spec.Parameters[i].Name < golden.FrameworkVersion.Spec.Parameters[j].Name
			})

			if diff := deep.Equal(golden, actual); diff != nil {
				t.Error(diff)
			}
		})
	}
}

func createTarball(tarballFile io.Writer, source string) error {
	gzipWriter := gzip.NewWriter(tarballFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.Walk(source,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}

			header.Name = strings.TrimPrefix(path, source)

			if err := tarWriter.WriteHeader(header); err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(tarWriter, file)
			return err
		})
}

func loadCrdsFromPath(goldenPath string) (*PackageCRDs, error) {
	result := &PackageCRDs{}
	err := filepath.Walk(goldenPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == goldenPath {
			// skip the root folder, as Walk always starts there
			return nil
		}
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		switch {
		case isFrameworkV0File(info.Name()):
			var f v1alpha1.Framework
			if err = yaml.Unmarshal(bytes, &f); err != nil {
				return errors.Wrapf(err, "cannot unmarshal %s content", info.Name())
			}
			result.Framework = &f
		case isVersionV0File(info.Name()):
			var fv v1alpha1.FrameworkVersion
			if err = yaml.Unmarshal(bytes, &fv); err != nil {
				return errors.Wrapf(err, "cannot unmarshal %s content", info.Name())
			}
			result.FrameworkVersion = &fv
		case isInstanceV0File(info.Name()):
			var i v1alpha1.Instance
			if err = yaml.Unmarshal(bytes, &i); err != nil {
				return errors.Wrapf(err, "cannot unmarshal %s content", info.Name())
			}
			result.Instance = &i
		default:
			return fmt.Errorf("unexpected file in the tarball structure %s", info.Name())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func isFrameworkV0File(name string) bool {
	return strings.HasSuffix(name, frameworkGolden)
}

func isVersionV0File(name string) bool {
	return strings.HasSuffix(name, versionGolden)
}

func isInstanceV0File(name string) bool {
	return strings.HasSuffix(name, instanceGolden)
}
