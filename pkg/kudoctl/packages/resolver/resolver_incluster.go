package resolver

import (
	"fmt"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/convert"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

// InClusterResolver resolves packages that are already installed in the cluster on the client-side. Note, that unlike
// other resolvers, the resulting 'packages.Package' struct does not contain package 'packages.Files' (we don't have
// the original files).
type InClusterResolver struct {
	c  *kudo.Client
	ns string
}

func (r InClusterResolver) Resolve(name string, appVersion string, operatorVersion string) (*packages.Package, error) {
	ovn := kudoapi.OperatorVersionName(name, operatorVersion)

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
