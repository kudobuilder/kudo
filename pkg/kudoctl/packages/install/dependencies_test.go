package install

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

func TestGatherDependencies(t *testing.T) {
	tests := []struct {
		name         string
		pkgs         []packages.Package
		expectedDeps []string
		expectedErr  string
	}{
		{
			// A <---> A
			name: "trivial circular dependency",
			pkgs: []packages.Package{
				createPackage("A", "A"),
			},
			expectedDeps: []string{},
			expectedErr:  "cyclic package dependency found when adding package A-:",
		},
		{
			// A <---> B
			name: "circular dependency",
			pkgs: []packages.Package{
				createPackage("A", "B"),
				createPackage("B", "A"),
			},
			expectedDeps: []string{},
			expectedErr:  "cyclic package dependency found when adding package B-:",
		},
		{
			// A ---> (B)
			name: "unknown dependency",
			pkgs: []packages.Package{
				createPackage("A", "B"),
			},
			expectedDeps: []string{},
			expectedErr:  "failed to resolve package B-:, dependency of package A-:: package not found",
		},
		{
			// A ---> B ---> C
			name: "simple dependency",
			pkgs: []packages.Package{
				createPackage("A", "B"),
				createPackage("B", "C"),
				createPackage("C"),
			},
			expectedDeps: []string{"B", "C"},
			expectedErr:  "",
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
			expectedDeps: []string{"C", "D", "E"},
			expectedErr:  "",
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
			expectedDeps: []string{},
			expectedErr:  "cyclic package dependency found when adding package B-:",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			resolver := nameResolver{tt.pkgs}
			actual, err := gatherDependencies(*tt.pkgs[0].Resources, resolver)

			assert.Equal(t, err == nil, tt.expectedErr == "")

			if err != nil {
				assert.EqualError(t, err, tt.expectedErr, tt.name)
			}

			for _, operatorName := range tt.expectedDeps {
				operatorName := operatorName

				assert.NotNil(t, funk.Find(actual, func(p packages.Resources) bool {
					return p.Operator.Name == operatorName
				}), tt.name)
			}
		})
	}
}
