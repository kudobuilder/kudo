package install

import (
	"strings"
	"time"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	deps "github.com/kudobuilder/kudo/pkg/kudoctl/resources/dependencies"
	"github.com/kudobuilder/kudo/pkg/kudoctl/resources/install"
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
	dependencies []deps.Dependency,
	options Options) error {
	clog.V(3).Printf(
		"Preparing %s/%s:%s for installation",
		namespace,
		resources.Operator.Name,
		resources.OperatorVersion.Spec.Version)

	applyOverrides(&resources, instanceName, namespace, parameters)

	if !options.SkipInstance {
		// If skipInstance is specified, we do not need to validate the parameters - If we do, we prevent the
		// installation of an operator version that has a parameter which is required but has no default.
		if err := validateParameters(
			*resources.Instance,
			resources.OperatorVersion.Spec.Parameters); err != nil {
			return err
		}
	}

	if err := client.ValidateServerForOperator(resources.Operator); err != nil {
		return err
	}

	if options.CreateNamespace {
		if err := installNamespace(client, resources, parameters); err != nil {
			return err
		}
	}

	if err := install.OperatorAndOperatorVersion(
		client, resources.Operator, resources.OperatorVersion, dependencies); err != nil {
		return err
	}

	if options.SkipInstance {
		return nil
	}

	if err := install.Instance(client, resources.Instance); err != nil {
		return err
	}

	if options.Wait != nil {
		if err := install.WaitForInstance(client, resources.Instance, *options.Wait); err != nil {
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

func validateParameters(instance kudoapi.Instance, parameters []kudoapi.Parameter) error {
	missingParameters := []string{}

	for _, p := range parameters {
		if p.IsRequired() && !p.HasDefault() {
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
