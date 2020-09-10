package instance

import (
	"fmt"
	"sort"

	"github.com/Masterminds/semver/v3"
	"github.com/thoas/go-funk"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

// InClusterResolver resolves packages that are already installed in the cluster. Note, that unlike other resolvers, the
// resulting 'packages.Package' struct does not contain package 'packages.Files' (we don't have the original files) and
// doesn't have an Instance resource because multiple Instances of the same Operator/OperatorVersion can exist
type InClusterResolver struct {
	c  client.Client
	ns string
}

type operatorVersionList []*kudoapi.OperatorVersion

// Len returns the number of entries
// This is needed to allow sorting.
func (b operatorVersionList) Len() int { return len(b) }

// Swap swaps the position of two items in the slice.
// This is needed to allow sorting.
func (b operatorVersionList) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

// Less returns true if the version of entry a is less than the version of entry b.
// This is needed to allow sorting.
func (b operatorVersionList) Less(x, y int) bool {

	// First compare Operator name
	if b[x].Spec.Operator.Name != b[y].Spec.Operator.Name {
		return b[x].Spec.Operator.Name < b[y].Spec.Operator.Name
	}

	// Compare OperatorVersion - Use SemVer if possible
	xVersion, _ := semver.NewVersion(b[x].Spec.Version)
	yVersion, _ := semver.NewVersion(b[y].Spec.Version)
	if xVersion != nil && yVersion != nil {
		res := xVersion.Compare(yVersion)
		if res != 0 {
			return res < 0
		}
	} else {
		if b[x].Spec.Version != b[y].Spec.Version {
			return b[x].Spec.Version < b[y].Spec.Version
		}
	}

	// Compare AppVersion - Use SemVer if possible
	xAppVersion, _ := semver.NewVersion(b[x].Spec.AppVersion)
	yAppVersion, _ := semver.NewVersion(b[y].Spec.AppVersion)
	if xAppVersion != nil && yAppVersion != nil {
		res := xAppVersion.Compare(yAppVersion)
		return res < 0
	}
	return b[x].Spec.AppVersion < b[y].Spec.AppVersion
}

func (r InClusterResolver) Resolve(name string, appVersion string, operatorVersion string) (*packages.Package, error) {
	ovList, err := kudoapi.ListOperatorVersions(r.c, r.ns)
	if err != nil {
		return nil, fmt.Errorf("failed to list operator versions in namespace %q: %v", r.ns, err)
	}

	// Put all items into a new list to be sortable
	var newOvList operatorVersionList
	for _, ovFromList := range ovList.Items {
		ovFromList := ovFromList
		newOvList = append(newOvList, &ovFromList)
	}

	// Only consider OVs for the given name
	// nolint:errcheck
	newOvList = funk.Filter(newOvList, func(ov *kudoapi.OperatorVersion) bool { return ov.Spec.Operator.Name == name }).(operatorVersionList)

	// Sort items
	// We want to pick the newest (highest) versions first, that's why we reverse the order
	sort.Sort(sort.Reverse(newOvList))

	// Find first matching OV
	var ov *kudoapi.OperatorVersion
	for _, ovFromList := range ovList.Items {
		ovFromList := ovFromList
		if name == ovFromList.Spec.Operator.Name &&
			(operatorVersion == "" || operatorVersion == ovFromList.Spec.Version) &&
			(appVersion == "" || appVersion == ovFromList.Spec.AppVersion) {
			ov = &ovFromList
			break
		}
	}
	if ov == nil {
		return nil, fmt.Errorf("failed to resolve operator version in namespace %q for name %q, version %q, appVersion %q", r.ns, name, operatorVersion, appVersion)
	}

	o, err := kudoapi.GetOperator(name, r.ns, r.c)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve operator %s/%s", r.ns, name)
	}

	return &packages.Package{
		Resources: &packages.Resources{
			Operator:        o,
			OperatorVersion: ov,
			Instance:        nil,
		},
		Files: nil,
	}, nil
}
