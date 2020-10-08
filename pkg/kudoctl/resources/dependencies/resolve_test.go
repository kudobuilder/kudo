package dependencies

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	engtask "github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

type nameResolver struct {
	Pkgs []packages.Package
}

func (resolver nameResolver) Resolve(
	name string,
	appVersion string,
	operatorVersion string) (*packages.Package, error) {
	for _, pkg := range resolver.Pkgs {
		if pkg.OperatorName() == name &&
			(operatorVersion == "" || pkg.OperatorVersionString() == operatorVersion) &&
			(appVersion == "" || pkg.AppVersionString() == appVersion) {
			return &pkg, nil
		}
	}

	return nil, fmt.Errorf("package not found")
}

func createPackage(name string, dependencies ...string) packages.Package {
	opVersion := "0.0.1"
	appVersion := ""

	deps := []kudoapi.Task{}
	for _, d := range dependencies {
		deps = append(deps, createDependency(d, "", ""))
	}

	return createPackageWithVersions(name, opVersion, appVersion, deps...)
}

func createDependency(name, opVersion, appVersion string) kudoapi.Task {
	return kudoapi.Task{
		Name: "dependency",
		Kind: engtask.KudoOperatorTaskKind,
		Spec: kudoapi.TaskSpec{
			KudoOperatorTaskSpec: kudoapi.KudoOperatorTaskSpec{
				Package:         name,
				OperatorVersion: opVersion,
				AppVersion:      appVersion,
			},
		},
	}
}

func createPackageWithVersions(name, opVersion, appVersion string, dependencies ...kudoapi.Task) packages.Package {
	p := packages.Package{
		Resources: &packages.Resources{
			Operator: &kudoapi.Operator{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
			},
			OperatorVersion: &kudoapi.OperatorVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: kudoapi.OperatorVersionName(name, appVersion, opVersion),
				},
				Spec: kudoapi.OperatorVersionSpec{
					Operator: v1.ObjectReference{
						Name: name,
					},
					Version:    opVersion,
					AppVersion: appVersion,
				},
			},
		},
	}

	p.Resources.OperatorVersion.Spec.Tasks = append(p.Resources.OperatorVersion.Spec.Tasks, dependencies...)

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
			wantErr: "cyclic package dependency found when adding package A-0.0.1 -> A-0.0.1",
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
			wantErr: "cyclic package dependency found when adding package B-0.0.1 -> B-0.0.1",
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
			wantErr: "cyclic package dependency found when adding package B-0.0.1 -> A-0.0.1",
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
			wantErr: "cyclic package dependency found when adding package C-0.0.1 -> B-0.0.1",
		},
		{
			// A
			// └── (B)
			name: "unknown dependency",
			pkgs: []packages.Package{
				createPackage("A", "B"),
			},
			want:    []string{},
			wantErr: "failed to resolve package Operator: \"B\", OperatorVersion: \"any\", AppVersion \"any\", dependency of package A-0.0.1: package not found",
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
			wantErr: "cyclic package dependency found when adding package C-0.0.1 -> A-0.0.1",
		},
		{
			// A
			// └── B
			name: "versioned dependency",
			pkgs: []packages.Package{
				createPackageWithVersions("A", "0.0.1", "1.2.3", createDependency("B", "0.0.2", "")),
				createPackageWithVersions("B", "0.0.1", ""),
			},
			want:    []string{},
			wantErr: "failed to resolve package Operator: \"B\", OperatorVersion: \"0.0.2\", AppVersion \"any\", dependency of package A-1.2.3-0.0.1: package not found",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			resolver := nameResolver{tt.pkgs}
			got, err := Resolve(tt.pkgs[0].Resources.OperatorVersion, resolver)

			assert.Equal(t, err == nil, tt.wantErr == "")

			if err != nil {
				assert.EqualError(t, err, tt.wantErr, tt.name)
			}

			for _, operatorName := range tt.want {
				operatorName := operatorName

				assert.NotNil(t, funk.Find(got, func(dep Dependency) bool {
					return dep.Operator.Name == operatorName
				}), fmt.Sprintf("failed to find wanted dependency %s", operatorName))
			}
		})
	}
}
