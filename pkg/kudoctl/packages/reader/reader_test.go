package reader

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

func TestReadFileSystemPackage(t *testing.T) {
	tests := []struct {
		name         string
		instanceName string
		path         string
		goldenFiles  string
	}{
		{"zookeeper", "zk1", "../testdata/zk", "../testdata/zk-crd-golden1"},
		{"zookeeper zipped", "zk2", "../testdata/zk.tgz", "../testdata/zk-crd-golden2"},
	}
	var fs = afero.NewOsFs()

	for _, tt := range tests {
		tt := tt

		t.Run(fmt.Sprintf("%s-from-%s", tt.name, tt.path), func(t *testing.T) {
			var err error
			var got *packages.Resources

			if strings.HasSuffix(tt.path, ".tgz") {
				got, err = ResourcesFromTar(fs, afero.NewMemMapFs(), tt.path)
			} else {
				got, err = ResourcesFromDir(fs, tt.path)
			}

			assert.NoError(t, err, "unexpected error while reading the package")

			got.Instance.ObjectMeta.Name = tt.instanceName
			golden, err := loadResourcesFromPath(tt.goldenFiles)
			if err != nil {
				t.Errorf("Found unexpected error when loading golden files: %v", err)
			}

			// we need to sort here because current yaml parsing is not preserving the order of fields
			// at the same time, the deep library we use for equality does not support ignoring order
			sort.Slice(got.OperatorVersion.Spec.Parameters, func(i, j int) bool {
				return got.OperatorVersion.Spec.Parameters[i].Name < got.OperatorVersion.Spec.Parameters[j].Name
			})
			sort.Slice(golden.OperatorVersion.Spec.Parameters, func(i, j int) bool {
				return golden.OperatorVersion.Spec.Parameters[i].Name < golden.OperatorVersion.Spec.Parameters[j].Name
			})

			assert.Equal(t, golden, got)
		})
	}
}

func loadResourcesFromPath(goldenPath string) (*packages.Resources, error) {
	isOperatorFile := func(name string) bool {
		return strings.HasSuffix(name, "operator.golden")
	}

	isVersionFile := func(name string) bool {
		return strings.HasSuffix(name, "operatorversion.golden")
	}

	isInstanceFile := func(name string) bool {
		return strings.HasSuffix(name, "instance.golden")
	}

	result := &packages.Resources{}
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
		case isOperatorFile(info.Name()):
			var f kudoapi.Operator
			if err = yaml.Unmarshal(bytes, &f); err != nil {
				return fmt.Errorf("cannot unmarshal %s content: %w", info.Name(), err)
			}
			result.Operator = &f
		case isVersionFile(info.Name()):
			var fv kudoapi.OperatorVersion
			if err = yaml.Unmarshal(bytes, &fv); err != nil {
				return fmt.Errorf("cannot unmarshal %s content: %w", info.Name(), err)
			}
			result.OperatorVersion = &fv
		case isInstanceFile(info.Name()):
			var i kudoapi.Instance
			if err = yaml.Unmarshal(bytes, &i); err != nil {
				return fmt.Errorf("cannot unmarshal %s content: %w", info.Name(), err)
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

func Test_readParametersFile(t *testing.T) {
	noParams := `
apiVersion: kudo.dev/v1beta1
parameters:
`
	oneParam := `
apiVersion: kudo.dev/v1beta1
parameters:
    - name: example
`
	example := []packages.Parameter{{Name: "example"}}

	tests := []struct {
		name       string
		paramsYaml string
		want       packages.ParamsFile
		wantErr    bool
	}{
		{"no data", "", packages.ParamsFile{APIVersion: APIVersion}, false},
		{"no parameters", noParams, packages.ParamsFile{APIVersion: APIVersion}, false},
		{"parameters", oneParam, packages.ParamsFile{APIVersion: APIVersion, Parameters: example}, false},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			got, err := readParametersFile([]byte(tt.paramsYaml))
			fmt.Printf("%s got: %v\n", tt.name, got)
			assert.Equal(t, tt.wantErr, err != nil, "readParametersFile() error = %v, wantErr %v", err, tt.wantErr)
			assert.Equal(t, tt.want, got)
		})
	}
}
