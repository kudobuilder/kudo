package dependencies

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	engtask "github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	pkgresolver "github.com/kudobuilder/kudo/pkg/kudoctl/packages/resolver"
)

type nameResolver struct {
	Prs []packages.Resources
}

func (resolver nameResolver) Resolve(
	name string,
	appVersion string,
	operatorVersion string) (*packages.PackageScope, error) {
	for _, pr := range resolver.Prs {
		pr := pr
		if pr.Operator.Name == name &&
			(operatorVersion == "" || pr.OperatorVersionString() == operatorVersion) &&
			(appVersion == "" || pr.AppVersionString() == appVersion) {
			return &packages.PackageScope{Resources: &pr, DependenciesResolver: resolver}, nil
		}
	}

	return nil, fmt.Errorf("package not found")
}

func createResources(name string, dependencies ...string) packages.Resources {
	opVersion := "0.0.1"
	appVersion := ""

	deps := []kudoapi.Task{}
	for _, d := range dependencies {
		deps = append(deps, createDependency(d, "", ""))
	}

	return createResourcesWithVersions(name, opVersion, appVersion, deps...)
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

func createResourcesWithVersions(name, opVersion, appVersion string, dependencies ...kudoapi.Task) packages.Resources {
	p := packages.Resources{
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
	}

	p.OperatorVersion.Spec.Tasks = append(p.OperatorVersion.Spec.Tasks, dependencies...)

	return p
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		prs     []packages.Resources
		want    []string
		wantErr string
	}{
		{
			// A
			// └── A
			name: "trivial circular dependency",
			prs: []packages.Resources{
				createResources("A", "A"),
			},
			want:    []string{},
			wantErr: "cyclic package dependency found when adding package A-0.0.1 -> A-0.0.1",
		},
		{
			// A
			// └── B
			//     └── B
			name: "trivial nested circular dependency",
			prs: []packages.Resources{
				createResources("A", "B"),
				createResources("B", "B"),
			},
			want:    []string{},
			wantErr: "cyclic package dependency found when adding package B-0.0.1 -> B-0.0.1",
		},
		{
			// A
			// └── B
			//     └── A
			name: "circular dependency",
			prs: []packages.Resources{
				createResources("A", "B"),
				createResources("B", "A"),
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
			prs: []packages.Resources{
				createResources("A", "B"),
				createResources("B", "C"),
				createResources("C", "B"),
			},
			want:    []string{},
			wantErr: "cyclic package dependency found when adding package C-0.0.1 -> B-0.0.1",
		},
		{
			// A
			// └── (B)
			name: "unknown dependency",
			prs: []packages.Resources{
				createResources("A", "B"),
			},
			want:    []string{},
			wantErr: "failed to resolve package Operator: \"B\", OperatorVersion: \"any\", AppVersion \"any\", dependency of package A-0.0.1: package not found",
		},
		{
			// A
			// └── B
			//     └── C
			name: "simple dependency",
			prs: []packages.Resources{
				createResources("A", "B"),
				createResources("B", "C"),
				createResources("C"),
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
			prs: []packages.Resources{
				createResources("A", "C", "D"),
				createResources("B", "C", "E"),
				createResources("C", "D"),
				createResources("D", "E"),
				createResources("E"),
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
			prs: []packages.Resources{
				createResources("A", "B"),
				createResources("B", "C", "E", "F"),
				createResources("C", "D", "A"),
				createResources("D"),
				createResources("E"),
				createResources("F"),
			},
			want:    []string{},
			wantErr: "cyclic package dependency found when adding package C-0.0.1 -> A-0.0.1",
		},
		{
			// A
			// └── B
			name: "versioned dependency",
			prs: []packages.Resources{
				createResourcesWithVersions("A", "0.0.1", "1.2.3", createDependency("B", "0.0.2", "")),
				createResourcesWithVersions("B", "0.0.1", ""),
			},
			want:    []string{},
			wantErr: "failed to resolve package Operator: \"B\", OperatorVersion: \"0.0.2\", AppVersion \"any\", dependency of package A-1.2.3-0.0.1: package not found",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			resolver := nameResolver{tt.prs}
			got, err := Resolve(tt.prs[0].OperatorVersion, resolver)

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

func TestResolveLocalDependencies(t *testing.T) {
	tests := []struct {
		path string
	}{
		{path: "./testdata/operator-with-dependencies/parent-operator"},
		{path: "./testdata/operator-with-dependencies.tgz"},
	}

	wd, _ := os.Getwd()

	var resolver = pkgresolver.NewPackageResolver(nil, wd)

	for _, tt := range tests {
		tt := tt
		t.Run(tt.path, func(t *testing.T) {
			pr, err := resolver.Resolve(tt.path, "", "")
			assert.NoError(t, err, "failed to resolve operator package for %s", tt.path)
			assert.NotNil(t, pr, "failed to resolve operator %s", tt.path)

			dependencies, err := Resolve(pr.Resources.OperatorVersion, pr.DependenciesResolver)
			assert.Equal(t, "child", pr.Resources.OperatorVersion.Spec.Tasks[0].Spec.KudoOperatorTaskSpec.Package, "expecting the KudoOperatorTask to have resolved child operator name")
			assert.Equal(t, "0.0.1", pr.Resources.OperatorVersion.Spec.Tasks[0].Spec.KudoOperatorTaskSpec.OperatorVersion, "expecting the KudoOperatorTask to have resolved child operator version")
			assert.Equal(t, "3.2.1", pr.Resources.OperatorVersion.Spec.Tasks[0].Spec.KudoOperatorTaskSpec.AppVersion, "expecting the KudoOperatorTask to have resolved child operator app version")

			assert.NoError(t, err, "failed to resolve dependencies for %s", tt.path)
			assert.Equal(t, len(dependencies), 1, "expecting to find child-operator dependency")

			assert.NotNil(t, dependencies[0].Operator, "expecting to find child-operator OperatorVersion resource")
			assert.NotNil(t, dependencies[0].OperatorVersion, "expecting to find child-operator OperatorVersion resource")
			assert.NotNil(t, dependencies[0].Instance, "expecting to find child-operator OperatorVersion resource")

			assert.Equal(t, dependencies[0].Operator.Name, "child")
			assert.Equal(t, dependencies[0].OperatorVersion.Name, kudoapi.OperatorVersionName("child", "3.2.1", "0.0.1"))
			assert.Equal(t, dependencies[0].Instance.Name, kudoapi.OperatorInstanceName("child"))
		})
	}
}
