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
			OperatorVersion: &v1beta1.OperatorVersion{},
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
		name        string
		pkgs        []packages.Package
		expected    []string
		expectedErr string
	}{
		{
			// A <---> A
			"trivial circular dependency",
			[]packages.Package{
				createPackage("A", "A"),
			},
			[]string{},
			"cyclic package dependency found when adding package A--",
		},
		{
			// A <---> B
			"circular dependency",
			[]packages.Package{
				createPackage("A", "B"),
				createPackage("B", "A"),
			},
			[]string{},
			"cyclic package dependency found when adding package A--",
		},
		{
			// A ---> (B)
			"unknown dependency",
			[]packages.Package{
				createPackage("A", "B"),
			},
			[]string{},
			"failed to resolve package B--, dependency of package A--: package not found",
		},
		{
			// A ---> B ---> C
			"simple dependency",
			[]packages.Package{
				createPackage("A", "B"),
				createPackage("B", "C"),
				createPackage("C"),
			},
			[]string{"A", "B", "C"},
			"",
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
			"complex dependency",
			[]packages.Package{
				createPackage("A", "C", "D"),
				createPackage("B", "C", "E"),
				createPackage("C", "D"),
				createPackage("D", "E"),
				createPackage("E"),
			},
			[]string{"A", "C", "D", "E"},
			"",
		},
	}

	for _, test := range tests {
		test := test

		resolver := nameResolver{test.pkgs}

		actual, err := gatherDependencies(*test.pkgs[0].Resources, resolver)
		if err != nil {
			if test.expectedErr == "" {
				t.Errorf("%s: expected no error but got %v", test.name, err)
			}

			assert.EqualError(t, err, test.expectedErr, test.name)
		} else {
			if test.expectedErr != "" {
				t.Errorf("%s: expected an error but got none", test.name)
			}

			for _, operatorName := range test.expected {
				operatorName := operatorName

				assert.NotNil(t, funk.Find(actual, func(p packages.Resources) bool {
					return p.Operator.Name == operatorName
				}), test.name)
			}
		}
	}
}
