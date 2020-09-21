package resolver

import (
	"fmt"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/convert"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	kudoutil "github.com/kudobuilder/kudo/pkg/util/kudo"
)

// InClusterResolver resolves packages that are already installed in the cluster on the client-side. Note, that unlike
// other resolvers, the resulting 'packages.Package' struct does not contain package 'packages.Files' (we don't have
// the original files).
type InClusterResolver struct {
	c  *kudo.Client
	ns string
}

func (r InClusterResolver) Resolve(name string, appVersion string, operatorVersion string) (*packages.Package, error) {
	ovList, err := r.c.ListOperatorVersions(r.ns)
	if err != nil {
		return nil, fmt.Errorf("failed to list operator versions in namespace %q: %v", r.ns, err)
	}

	if len(ovList) == 0 {
		return nil, fmt.Errorf("failed to find any operator version in namespace %s", r.ns)
	}

	// Put all items into a new list to be sortable
	newOvList := kudoutil.SortableOperatorList{}
	for _, ovFromList := range ovList {
		ovFromList := ovFromList
		newOvList = append(newOvList, &ovFromList)
	}

	// Only consider OVs for the given name
	newOvList = newOvList.FilterByName(name)

	// Sort items
	newOvList.Sort()

	// Find first matching OV
	ov, _ := newOvList.FindFirstMatch(name, operatorVersion, appVersion).(*kudoapi.OperatorVersion) // nolint:errcheck

	if ov == nil {
		return nil, fmt.Errorf("failed to resolve operator version in namespace %q for name %q, version %q, appVersion %q", r.ns, name, operatorVersion, appVersion)
	}
	o, err := r.c.GetOperator(name, r.ns)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve operator %s/%s", r.ns, name)
	}

	return &packages.Package{
		Resources: &packages.Resources{
			Operator:        o,
			OperatorVersion: ov,
			Instance:        convert.BuildInstanceResource(name, operatorVersion, appVersion),
		},
		Files: nil,
	}, nil
}
