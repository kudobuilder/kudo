package install

import (
	"fmt"

	"github.com/thoas/go-funk"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	engtask "github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/resolver"
	deps "github.com/kudobuilder/kudo/pkg/kudoctl/resources/dependencies"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

// OperatorAndOperatorVersion installs both of these object to a cluster.
// Operators can contain dependencies on other operators. In this case
// the O/OV of dependencies are installed as well.
func OperatorAndOperatorVersion(
	client *kudo.Client,
	operator *v1beta1.Operator,
	operatorVersion *v1beta1.OperatorVersion,
	resolver resolver.Resolver) error {
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
		if err := installDependencies(client, operatorVersion, resolver); err != nil {
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

func installDependencies(client *kudo.Client, ov *v1beta1.OperatorVersion, resolver resolver.Resolver) error {
	dependencies, err := deps.Resolve(ov, resolver)
	if err != nil {
		return err
	}

	// The KUDO controller will create Instances for the dependencies. For this
	// it needs to resolve the dependencies again from 'KudoOperatorTaskSpec'.
	// But it cannot resolve packages like the CLI, because it may
	// not have access to the referenced local files or URLs.
	// It can however resolve the OperatorVersion from the name of the operator
	// dependency. For this, we overwrite the 'Package' field describing
	// dependencies in 'KudoOperatorTaskSpec' with the operator name of the
	// dependency. This has to be done for the operator to upgrade as well as in
	// all of its new dependencies.

	updateKudoOperatorTaskPackageNames(dependencies, ov)

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
			updateKudoOperatorTaskPackageNames(dependencies, dependency.OperatorVersion)

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

// updateKudoOperatorTaskPackageNames sets the 'Package' and 'OperatorName'
// fields of the 'KudoOperatorTaskSpec' of an 'OperatorVersion' to the operator name
// initially referenced in the 'Package' field.
func updateKudoOperatorTaskPackageNames(
	dependencies []deps.Dependency, operatorVersion *v1beta1.OperatorVersion) {
	tasks := operatorVersion.Spec.Tasks

	for i := range tasks {
		if tasks[i].Kind == engtask.KudoOperatorTaskKind {
			for _, dependency := range dependencies {
				if tasks[i].Spec.KudoOperatorTaskSpec.Package == dependency.PackageName {
					tasks[i].Spec.KudoOperatorTaskSpec.Package = dependency.Operator.Name
					tasks[i].Spec.KudoOperatorTaskSpec.OperatorVersion = dependency.OperatorVersion.Spec.Version
					break
				}
			}
		}
	}

	operatorVersion.Spec.Tasks = tasks
}
