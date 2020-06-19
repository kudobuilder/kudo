package install

import (
	"strings"
	"time"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	engtask "github.com/kudobuilder/kudo/pkg/engine/task"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/resolver"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

type Options struct {
	// Skip instance resource creation.
	SkipInstance bool
	// Wait until the instance has been created.
	Wait *time.Duration
	// Create the namespace for the operator package.
	CreateNamespace bool
}

// Package installs an operator package with parameters into a namespace.
// Instance name, namespace and operator parameters are applied to the
// operator package resources. These rendered resources are then created
// on the Kubernetes cluster.
// Packages can have dependencies on other packages. In that case,
// dependent packages are resolved and their Operator and
// Operatorversion resources created on the Kubernetes cluster.
func Package(
	client *kudo.Client,
	instanceName string,
	namespace string,
	resources packages.Resources,
	parameters map[string]string,
	resolver resolver.Resolver,
	options Options) error {
	clog.V(3).Printf(
		"Preparing %s/%s:%s for installation",
		namespace,
		resources.Operator.Name,
		resources.OperatorVersion.Spec.Version)

	applyOverrides(&resources, instanceName, namespace, parameters)

	if err := validateParameters(
		*resources.Instance,
		resources.OperatorVersion.Spec.Parameters); err != nil {
		return err
	}

	if err := client.ValidateServerForOperator(resources.Operator); err != nil {
		return err
	}

	if options.CreateNamespace {
		if err := installNamespace(client, resources, parameters); err != nil {
			return err
		}
	}

	dependencies, err := ResolveDependencies(resources, resolver)
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
	// dependency. This has to be done for the operator to install as well as in
	// all of its dependencies.

	updateKudoOperatorTaskPackageNames(dependencies, resources.OperatorVersion)

	for _, dependency := range dependencies {
		dependency.Operator.SetNamespace(namespace)
		dependency.OperatorVersion.SetNamespace(namespace)

		updateKudoOperatorTaskPackageNames(dependencies, dependency.OperatorVersion)

		if err := installOperatorAndOperatorVersion(client, dependency.Resources); err != nil {
			return err
		}
	}

	if err := installOperatorAndOperatorVersion(client, resources); err != nil {
		return err
	}

	if options.SkipInstance {
		return nil
	}

	if err := installInstance(client, resources.Instance); err != nil {
		return err
	}

	if options.Wait != nil {
		if err := waitForInstance(client, resources.Instance, *options.Wait); err != nil {
			return err
		}
	}

	return nil
}

func applyOverrides(
	resources *packages.Resources,
	instanceName string,
	namespace string,
	parameters map[string]string) {
	resources.Operator.SetNamespace(namespace)
	resources.OperatorVersion.SetNamespace(namespace)
	resources.Instance.SetNamespace(namespace)

	if instanceName != "" {
		clog.V(3).Printf(
			"Overriding instance name %s/%s to %s/%s",
			namespace,
			resources.Instance.Name,
			namespace,
			instanceName)
		resources.Instance.SetName(instanceName)
	}

	if parameters != nil {
		clog.V(3).Printf("parameters in use: %v", parameters)
		resources.Instance.Spec.Parameters = parameters
	}
}

func validateParameters(instance v1beta1.Instance, parameters []v1beta1.Parameter) error {
	missingParameters := []string{}

	for _, p := range parameters {
		if *p.Required && p.Default == nil {
			_, ok := instance.Spec.Parameters[p.Name]
			if !ok {
				missingParameters = append(missingParameters, p.Name)
			}
		}
	}

	if len(missingParameters) > 0 {
		return clog.Errorf(
			"missing required parameters during installation: %s",
			strings.Join(missingParameters, ","))
	}

	return nil
}

// updateKudoOperatorTaskPackageNames sets the 'Package' and 'OperatorName'
// fields of the 'KudoOperatorTaskSpec' of an 'OperatorVersion' to the operator name
// initially referenced in the 'Package' field.
func updateKudoOperatorTaskPackageNames(pkgs []Dependency, operatorVersion *v1beta1.OperatorVersion) {
	tasks := operatorVersion.Spec.Tasks

	for i := range tasks {
		if tasks[i].Kind == engtask.KudoOperatorTaskKind {
			for _, pkg := range pkgs {
				if tasks[i].Spec.KudoOperatorTaskSpec.Package == pkg.PackageName {
					tasks[i].Spec.KudoOperatorTaskSpec.Package = pkg.Operator.Name
					break
				}
			}
		}
	}

	operatorVersion.Spec.Tasks = tasks
}
