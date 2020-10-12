package dependencies

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	engtask "github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	pkgresolver "github.com/kudobuilder/kudo/pkg/kudoctl/packages/resolver"
)

type nameResolver struct {
	Pkgs []packages.Package
}

func (resolver nameResolver) Resolve(
	name string,
	appVersion string,
	operatorVersion string) (*packages.Package, error) {
	for _, pkg := range resolver.Pkgs {
		if pkg.Resources.Operator.Name == name {
			return &pkg, nil
		}
	}

	return nil, fmt.Errorf("package not found")
}

func createPackage(name string, dependencies ...string) packages.Package {
	p := packages.Package{
		Resources: &packages.Resources{
			Operator: &kudoapi.Operator{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
			},
			OperatorVersion: &kudoapi.OperatorVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: kudoapi.OperatorVersionName(name, ""),
				},
			},
		},
	}

	for _, dependency := range dependencies {
		p.Resources.OperatorVersion.Spec.Tasks = append(
			p.Resources.OperatorVersion.Spec.Tasks,
			kudoapi.Task{
				Name: "dependency",
				Kind: engtask.KudoOperatorTaskKind,
				Spec: kudoapi.TaskSpec{
					KudoOperatorTaskSpec: kudoapi.KudoOperatorTaskSpec{
						Package: dependency,
					},
				},
			})
	}

	return p
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		pkgs    []packages.Package
		want    []string
		wantErr string
	}{
		{
			// A
			// └── A
			name: "trivial circular dependency",
			pkgs: []packages.Package{
				createPackage("A", "A"),
			},
			want:    []string{},
			wantErr: "cyclic package dependency found when adding package A-- -> A--",
		},
		{
			// A
			// └── B
			//     └── B
			name: "trivial nested circular dependency",
			pkgs: []packages.Package{
				createPackage("A", "B"),
				createPackage("B", "B"),
			},
			want:    []string{},
			wantErr: "cyclic package dependency found when adding package B-- -> B--",
		},
		{
			// A
			// └── B
			//     └── A
			name: "circular dependency",
			pkgs: []packages.Package{
				createPackage("A", "B"),
				createPackage("B", "A"),
			},
			want:    []string{},
			wantErr: "cyclic package dependency found when adding package B-- -> A--",
		},
		{
			// A
			// └── B
			//     └── C
			//     	   └── B
			name: "nested circular dependency",
			pkgs: []packages.Package{
				createPackage("A", "B"),
				createPackage("B", "C"),
				createPackage("C", "B"),
			},
			want:    []string{},
			wantErr: "cyclic package dependency found when adding package C-- -> B--",
		},
		{
			// A
			// └── (B)
			name: "unknown dependency",
			pkgs: []packages.Package{
				createPackage("A", "B"),
			},
			want:    []string{},
			wantErr: "failed to resolve package B--, dependency of package A--: package not found",
		},
		{
			// A
			// └── B
			//     └── C
			name: "simple dependency",
			pkgs: []packages.Package{
				createPackage("A", "B"),
				createPackage("B", "C"),
				createPackage("C"),
			},
			want:    []string{"B", "C"},
			wantErr: "",
		},
		{
			//        B -----
			//        |      \
			//        |      |
			//        v      |
			// A ---> C      |
			// |      |      |
			// |      |      |
			// \      v      v
			//  ----> D ---> E
			name: "complex dependency",
			pkgs: []packages.Package{
				createPackage("A", "C", "D"),
				createPackage("B", "C", "E"),
				createPackage("C", "D"),
				createPackage("D", "E"),
				createPackage("E"),
			},
			want:    []string{"C", "D", "E"},
			wantErr: "",
		},
		{
			// A
			// └── B
			//     ├── C
			//     │   ├── D
			//     │   └── A
			//     ├── E
			//     └── F
			name: "complex circular dependency",
			pkgs: []packages.Package{
				createPackage("A", "B"),
				createPackage("B", "C", "E", "F"),
				createPackage("C", "D", "A"),
				createPackage("D"),
				createPackage("E"),
				createPackage("F"),
			},
			want:    []string{},
			wantErr: "cyclic package dependency found when adding package C-- -> A--",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			resolver := nameResolver{tt.pkgs}
			operatorArg := tt.pkgs[0].Resources.OperatorVersion.Name
			got, err := Resolve(operatorArg, tt.pkgs[0].Resources.OperatorVersion, resolver)

			assert.Equal(t, err == nil, tt.wantErr == "")

			if err != nil {
				assert.EqualError(t, err, tt.wantErr, tt.name)
			}

			for _, operatorName := range tt.want {
				operatorName := operatorName

				assert.NotNil(t, funk.Find(got, func(dep Dependency) bool {
					return dep.Operator.Name == operatorName
				}), tt.name)
			}
		})
	}
}

func Test_isLocalDirPackage(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantNilPath bool
		wantErr     bool
	}{
		{
			name:        "./",
			path:        "./",
			wantNilPath: false,
			wantErr:     false,
		},
		{
			name:        "../install",
			path:        "../install",
			wantNilPath: false,
			wantErr:     false,
		},
		{
			name:        "./some-fake-path",
			path:        "./some-fake-path",
			wantNilPath: true,
			wantErr:     true,
		},
		{
			name:        "./",
			path:        "./resolve_test.go",
			wantNilPath: true,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			absPath, err := operatorAbsPath(tt.path)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantNilPath, absPath == nil)
		})
	}
}

func TestResolveLocalDependencies(t *testing.T) {
	var resolver = pkgresolver.NewLocal()
	var operatorArgument = "./testdata/operator-with-dependencies/parent-operator"

	pkg, err := resolver.Resolve(operatorArgument, "", "")
	assert.NoError(t, err, "failed to resolve operator package for %s", operatorArgument)

	dependencies, err := Resolve(operatorArgument, pkg.Resources.OperatorVersion, resolver)
	assert.NoError(t, err, "failed to resolve dependencies for %s", operatorArgument)
	assert.Equal(t, len(dependencies), 1, "expecting to find child-operator dependency")

	assert.NotNil(t, dependencies[0].Operator, "expecting to find child-operator OperatorVersion resource")
	assert.NotNil(t, dependencies[0].OperatorVersion, "expecting to find child-operator OperatorVersion resource")
	assert.NotNil(t, dependencies[0].Instance, "expecting to find child-operator OperatorVersion resource")

	assert.Equal(t, dependencies[0].Operator.Name, "child")
	assert.Equal(t, dependencies[0].OperatorVersion.Name, kudoapi.OperatorVersionName("child", "0.0.1"))
	assert.Equal(t, dependencies[0].Instance.Name, kudoapi.OperatorInstanceName("child"))

	s, _ := json.MarshalIndent(dependencies, "", "  ")
	log.Printf(string(s))
}
