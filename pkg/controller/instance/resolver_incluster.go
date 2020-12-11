package instance

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

// InClusterResolver is a server-side package resolver for packages that are already installed in the cluster. It is a simpler
// version of the client-side pkg/kudoctl/packages/resolver/resolver_incluster.go. The client-side version would search
// the installed OperatorVersions and try to resolve any valid combination of the operator name and its app and operator versions,
// same as we would search in the repository.
// This resolver is only used to make sure that all the dependencies of an operator exist and that referenced operator versions
// are installed and uniquely identifiable by the passed operator name, appVersion and operatorVersion parameters
// (see pkg/apis/kudo/v1beta1/operatorversion_types_helpers.go::OperatorVersionName method).
type InClusterResolver struct {
	c  client.Client
	ns string
}

func NewInClusterResolver(client client.Client, ns string) *InClusterResolver {
	return &InClusterResolver{
		c:  client,
		ns: ns,
	}
}

func (r InClusterResolver) Resolve(name string, appVersion string, operatorVersion string) (*packages.PackageScope, error) {
	ovn := kudoapi.OperatorVersionName(name, appVersion, operatorVersion)

	ov, err := kudoapi.GetOperatorVersionByName(ovn, r.ns, r.c)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve operator version %s/%s:%s", r.ns, ovn, appVersion)
	}

	o, err := kudoapi.GetOperator(name, r.ns, r.c)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve operator %s/%s", r.ns, name)
	}

	res := &packages.Resources{
		Operator:        o,
		OperatorVersion: ov,
		Instance:        nil,
	}

	return &packages.PackageScope{Resources: res, DependenciesResolver: r}, nil
}
