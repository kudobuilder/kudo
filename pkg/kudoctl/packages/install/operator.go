package install

import (
	"fmt"

	"github.com/thoas/go-funk"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

func installOperatorAndOperatorVersion(client *kudo.Client, resources packages.Resources) error {
	if !client.OperatorExistsInCluster(resources.Operator.Name, resources.Operator.Namespace) {
		if _, err := client.InstallOperatorObjToCluster(resources.Operator, resources.Operator.Namespace); err != nil {
			return fmt.Errorf(
				"failed to install %s-operator.yaml in namespace %s: %v",
				resources.Operator.Name,
				resources.Operator.Namespace,
				err)
		}
		clog.Printf(
			"operator.%s/%s created in namespace %s",
			resources.Operator.APIVersion,
			resources.Operator.Name,
			resources.Operator.Namespace)
	}

	versionsInstalled, err := client.OperatorVersionsInstalled(resources.Operator.Name, resources.Operator.Namespace)
	if err != nil {
		return fmt.Errorf(
			"failed to retrieve existing operator versions of operator.%s/%s in namespace %s: %v",
			resources.Operator.APIVersion,
			resources.Operator.Name,
			resources.Operator.Namespace,
			err)
	}

	if !funk.ContainsString(versionsInstalled, resources.OperatorVersion.Spec.Version) {
		if _, err := client.InstallOperatorVersionObjToCluster(
			resources.OperatorVersion,
			resources.OperatorVersion.Namespace); err != nil {
			return fmt.Errorf(
				"failed to install %s-operatorversion.yaml in namespace %s: %v",
				resources.OperatorVersion.Name,
				resources.OperatorVersion.Namespace,
				err)
		}
		clog.Printf(
			"operatorversion.%s/%s created in namespace %s",
			resources.OperatorVersion.APIVersion,
			resources.OperatorVersion.Name,
			resources.OperatorVersion.Namespace)
	} else {
		clog.Printf(
			"operatorversion.%s/%s already installed in namespace %s",
			resources.OperatorVersion.APIVersion,
			resources.OperatorVersion.Name,
			resources.OperatorVersion.Namespace)
	}

	return nil
}
