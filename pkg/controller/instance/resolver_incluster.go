package instance

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

// InClusterResolver resolves packages that are already installed in the cluster. Note, that unlike other resolvers, the
// resulting 'packages.Package' struct does not contain package 'packages.Files' (we don't have the original files) and
// doesn't have an Instance resource because multiple Instances of the same Operator/OperatorVersion can exist
type InClusterResolver struct {
	c  client.Client
	ns string
}

func (r InClusterResolver) Resolve(name string, appVersion string, operatorVersion string) (*packages.Package, error) {
	ovn := v1beta1.OperatorVersionName(name, operatorVersion)

	ov, err := v1beta1.GetOperatorVersionByName(ovn, r.ns, r.c)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve operator version %s/%s:%s", r.ns, ovn, appVersion)
	}

	// sanity check, as there is an explicit 1:1 relationship between an operator and app version
	if ov.Spec.AppVersion != appVersion {
		return nil, fmt.Errorf("found operator version %s/%s but found appVersion %s is not equal to the requested %s", r.ns, ovn, ov.Spec.AppVersion, appVersion)
	}

	o, err := v1beta1.GetOperator(name, r.ns, r.c)
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
