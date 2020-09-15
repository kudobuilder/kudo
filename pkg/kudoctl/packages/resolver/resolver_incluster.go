package resolver

import (
	"fmt"
	"sort"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/convert"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
)

// InClusterResolver resolves packages that are already installed in the cluster on the client-side. Note, that unlike
// other resolvers, the resulting 'packages.Package' struct does not contain package 'packages.Files' (we don't have
// the original files).
type InClusterResolver struct {
	c  *kudo.Client
	ns string
}

func (r InClusterResolver) Resolve(name string, appVersion string, operatorVersion string) (*packages.Package, error) {
	// 1. find all in-cluster operator versions with the passed operator name
	versions, err := r.FindInClusterOperatorVersions(name)
	if err != nil {
		return nil, err
	}

	//2.  sorting packages in descending order same as the repo does it: pkg/kudoctl/util/repo/index.go::sortPackages
	// to preserve the selection rules. See sortPackages method description for more details.
	sort.Sort(sort.Reverse(versions))

	// 3. find first matching operator version
	version, err := repo.FindFirstMatchForEntries(versions, name, appVersion, operatorVersion)
	if err != nil {
		return nil, err
	}

	// 4. fetch the existing O/OV and install the instance
	ovn := version.Name
	operatorVersion = version.OperatorVersion

	ov, err := r.c.GetOperatorVersion(ovn, r.ns)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve operator version %s/%s:%s", r.ns, ovn, appVersion)
	}

	o, err := r.c.GetOperator(name, r.ns)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve operator %s/%s", r.ns, name)
	}

	i := convert.BuildInstanceResource(name, operatorVersion)

	return &packages.Package{
		Resources: &packages.Resources{
			Operator:        o,
			OperatorVersion: ov,
			Instance:        i,
		},
		Files: nil,
	}, nil
}

// FindInClusterOperatorVersions method searches for all in-cluster operator versions for the passed operator name
// and returns them as an []*PackageVersion array
func (r InClusterResolver) FindInClusterOperatorVersions(operatorName string) (repo.PackageVersions, error) {
	ovs, err := r.c.ListOperatorVersions(r.ns)
	if err != nil {
		return nil, fmt.Errorf("failed to list in-cluster operator %s versions: %v", operatorName, err)
	}

	versions := repo.PackageVersions{}
	for _, ov := range ovs {
		if ov.Spec.Operator.Name == operatorName {
			versions = append(versions, &repo.PackageVersion{
				Metadata: &repo.Metadata{
					Name:            ov.Name,
					OperatorVersion: ov.Spec.Version,
					AppVersion:      ov.Spec.AppVersion,
				},
			})
		}
	}

	return versions, nil
}
