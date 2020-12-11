package install

import (
	"fmt"

	"github.com/thoas/go-funk"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	deps "github.com/kudobuilder/kudo/pkg/kudoctl/resources/dependencies"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

// OperatorAndOperatorVersion installs both of these object to a cluster.
// Operators can contain dependencies on other operators. In this case
// the O/OV of dependencies are installed as well.
func OperatorAndOperatorVersion(
	client *kudo.Client,
	operator *kudoapi.Operator,
	operatorVersion *kudoapi.OperatorVersion,
	dependencies []deps.Dependency) error {
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
		if err := installDependencies(client, operatorVersion, dependencies); err != nil {
			return fmt.Errorf(
				"failed to install dependencies of operatorversion %s/%s: %v",
				operatorVersion.Namespace,
				operatorVersion.Name,
				err)
		}

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

func installDependencies(client *kudo.Client, ov *kudoapi.OperatorVersion, dependencies []deps.Dependency) error {
	for _, dependency := range dependencies {
		dependency.Operator.SetNamespace(ov.Namespace)
		dependency.OperatorVersion.SetNamespace(ov.Namespace)

		if !client.OperatorExistsInCluster(dependency.Operator.Name, dependency.Operator.Namespace) {
			if _, err := client.InstallOperatorObjToCluster(dependency.Operator, dependency.Operator.Namespace); err != nil {
				return fmt.Errorf(
					"failed to install operator %s/%s: %v",
					dependency.Operator.Namespace,
					dependency.Operator.Name,
					err)
			}
			clog.Printf("operator %s/%s created", dependency.Operator.Namespace, dependency.Operator.Name)
		}

		installed, err := client.OperatorVersionsInstalled(dependency.Operator.Name, dependency.Operator.Namespace)
		if err != nil {
			return fmt.Errorf(
				"failed to retrieve operatorversion of dependency %s/%s: %v",
				dependency.Operator.Namespace,
				dependency.Operator.Name,
				err)
		}

		if !funk.ContainsString(installed, dependency.OperatorVersion.Spec.Version) {
			if _, err := client.InstallOperatorVersionObjToCluster(
				dependency.OperatorVersion,
				dependency.OperatorVersion.Namespace); err != nil {
				return fmt.Errorf(
					"failed to install operatorversion %s/%s: %v",
					dependency.OperatorVersion.Namespace,
					dependency.OperatorVersion.Name,
					err)
			}
			clog.Printf(
				"operatorversion %s/%s created",
				dependency.OperatorVersion.Namespace,
				dependency.OperatorVersion.Name)
		}
	}

	return nil
}
