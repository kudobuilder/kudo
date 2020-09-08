package instance

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

// InClusterResolver resolves packages that are already installed in the cluster on the server-side. Note, that unlike
// other resolvers, the resulting 'packages.Package' struct does not contain package 'packages.Files' (we don't have
// the original files) and doesn't have an Instance resource because multiple Instances of the same Operator/OperatorVersion
// can exist.
type InClusterResolver struct {
	c  client.Client
	ns string
}

func (r InClusterResolver) Resolve(name string, appVersion string, operatorVersion string) (*packages.Package, error) {
	ovn := kudoapi.OperatorVersionName(name, operatorVersion)

	ov, err := kudoapi.GetOperatorVersionByName(ovn, r.ns, r.c)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve operator version %s/%s:%s", r.ns, ovn, appVersion)
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
