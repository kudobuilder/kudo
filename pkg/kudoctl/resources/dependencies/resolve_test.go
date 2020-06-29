package dependencies

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
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
		if pkg.Resources.Operator.Name == name {
			return &pkg, nil
		}
	}

	return nil, fmt.Errorf("package not found")
}

func createPackage(name string, dependencies ...string) packages.Package {
	p := packages.Package{
		Resources: &packages.Resources{
			Operator: &v1beta1.Operator{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
			},
			OperatorVersion: &v1beta1.OperatorVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: v1beta1.OperatorVersionName(name, ""),
				},
			},
		},
	}

	for _, dependency := range dependencies {
		p.Resources.OperatorVersion.Spec.Tasks = append(
			p.Resources.OperatorVersion.Spec.Tasks,
			v1beta1.Task{
				Name: "dependency",
				Kind: engtask.KudoOperatorTaskKind,
				Spec: v1beta1.TaskSpec{
					KudoOperatorTaskSpec: v1beta1.KudoOperatorTaskSpec{
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
			got, err := Resolve(tt.pkgs[0].Resources.OperatorVersion, resolver)

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
