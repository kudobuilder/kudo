package install

import (
	"fmt"

	"github.com/thoas/go-funk"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

// OperatorAndOperatorVersion installs both of these object to a cluster.
func OperatorAndOperatorVersion(
	client *kudo.Client,
	operator *v1beta1.Operator,
	operatorVersion *v1beta1.OperatorVersion) error {
	if !client.OperatorExistsInCluster(operator.Name, operator.Namespace) {
		if _, err := client.InstallOperatorObjToCluster(operator, operator.Namespace); err != nil {
			return fmt.Errorf(
				"failed to install operator %s/%s: %v",
				operator.Namespace,
				operator.Name,
				err)
		}
		clog.Printf("operator %s/%s created", operator.Namespace, operator.Name)
	}

	versionsInstalled, err := client.OperatorVersionsInstalled(operator.Name, operator.Namespace)
	if err != nil {
		return fmt.Errorf(
			"failed to retrieve existing operator versions of operator %s/%s: %v",
			operator.Namespace,
			operator.Name,
			err)
	}

	if !funk.ContainsString(versionsInstalled, operatorVersion.Spec.Version) {
		if _, err := client.InstallOperatorVersionObjToCluster(
			operatorVersion,
			operatorVersion.Namespace); err != nil {
			return fmt.Errorf(
				"failed to install operatorversion %s/%s: %v",
				operatorVersion.Namespace,
				operatorVersion.Name,
				err)
		}
		clog.Printf("operatorversion %s/%s created", operatorVersion.Namespace, operatorVersion.Name)
	} else {
		clog.Printf("operatorversion %s/%s already installed", operatorVersion.Namespace, operatorVersion.Name)
	}

	return nil
}
