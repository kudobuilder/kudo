package repo

import (
	"fmt"
	"github.com/go-test/deep"
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"sort"
	"testing"
)

func TestConvertV1Package(t *testing.T) {
	tests := []struct {
		name string
		v1PackageFolder string
		outputGoldenFolder string
	}{
		{"zookeeper", "testdata/zk", "testdata/zk-crd-golden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := PackageFromFileSystem(tt.v1PackageFolder)
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

func loadCrdsFromPath(goldenPath string) (*InstallCRDs, error) {
	result := &InstallCRDs{}
	err := filepath.Walk(goldenPath, func(path string, info os.FileInfo, err error) error {
		if path == goldenPath {
			// skip the root folder, as Walk always starts there
			return nil
		}
		bytes, err := ioutil.ReadFile(path)
		switch {
		case isFrameworkV0File(info.Name()):
			var f v1alpha1.Framework
			if err = yaml.Unmarshal(bytes, &f); err != nil {
				return errors.Wrapf(err, "unmarshalling %s content", info.Name())
			}
			result.Framework = &f
		case isVersionV0File(info.Name()):
			var fv v1alpha1.FrameworkVersion
			if err = yaml.Unmarshal(bytes, &fv); err != nil {
				return errors.Wrapf(err, "unmarshalling %s content", info.Name())
			}
			result.FrameworkVersion = &fv
		case isInstanceV0File(info.Name()):
			var i v1alpha1.Instance
			if err = yaml.Unmarshal(bytes, &i); err != nil {
				return errors.Wrapf(err, "unmarshalling %s content", info.Name())
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