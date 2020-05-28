// Package install provides function to install package resources
// on a Kubernetes cluster.
package install

import (
	"strings"
	"time"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

type Options struct {
	skipInstance    bool
	wait            *time.Duration
	createNamespace bool
}

type Option func(*Options)

// SkipInstance installs only Operator and OperatorVersion
// of an operator package.
func SkipInstance() Option {
	return func(o *Options) {
		o.skipInstance = true
	}
}

// WaitForInstance waits an amount of time for the instance
// to complete installation.
func WaitForInstance(duration time.Duration) Option {
	return func(o *Options) {
		o.wait = &duration
	}
}

// CreateNamespace creates the specified namespace before installation.
// If available, a namespace manifest in the operator package is
// rendered using the installation parameters.
func CreateNamespace() Option {
	return func(o *Options) {
		o.createNamespace = true
	}
}

// Package installs an operator package with parameters into a namespace.
// Instance name, namespace and operator parameters are applied to the
// operator package resources. These rendered resources are then created
// on the Kubernetes cluster.
func Package(
	client *kudo.Client,
	instanceName string,
	namespace string,
	resources packages.Resources,
	parameters map[string]string,
	opts ...Option) error {
	clog.V(3).Printf("operator name: %v", resources.Operator.Name)
	clog.V(3).Printf("operator version: %v", resources.OperatorVersion.Spec.Version)

	options := Options{}
	for _, o := range opts {
		o(&options)
	}

	applyOverrides(&resources, instanceName, namespace, parameters)

	if err := client.ValidateServerForOperator(resources.Operator); err != nil {
		return err
	}

	if options.createNamespace {
		if err := installNamespace(client, resources, parameters); err != nil {
			return err
		}
	}

	if err := installOperatorAndOperatorVersion(client, resources); err != nil {
		return err
	}

	if options.skipInstance {
		return nil
	}

	if err := validateParameters(
		*resources.Instance,
		resources.OperatorVersion.Spec.Parameters); err != nil {
		return err
	}

	if err := installInstance(client, resources.Instance); err != nil {
		return err
	}

	if options.wait != nil {
		if err := waitForInstance(client, resources.Instance, *options.wait); err != nil {
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
		resources.Instance.SetName(instanceName)
		clog.V(3).Printf("instance name: %v", instanceName)
	}
	if parameters != nil {
		resources.Instance.Spec.Parameters = parameters
		clog.V(3).Printf("parameters in use: %v", parameters)
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
