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
	Client    *kudo.Client
	Namespace string
}

func (r InClusterResolver) Resolve(name string, appVersion string, operatorVersion string) (*packages.Package, error) {
	ovn := kudoapi.OperatorVersionName(name, operatorVersion)

	ov, err := r.Client.GetOperatorVersion(ovn, r.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve operator version %s/%s:%s", r.Namespace, ovn, appVersion)
	}

	o, err := r.Client.GetOperator(name, r.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve operator %s/%s", r.Namespace, name)
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
