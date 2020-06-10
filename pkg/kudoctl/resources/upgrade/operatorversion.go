package upgrade

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/thoas/go-funk"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/dependencies"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages/resolver"
	"github.com/kudobuilder/kudo/pkg/kudoctl/resources/install"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/util/convert"
)

// OperatorVersion upgrades an OperatorVersion and its Instance.
// For the updated Instance, new parameters can be provided.
func OperatorVersion(kc *kudo.Client, newOv *v1beta1.OperatorVersion, instanceName, namespace string, parameters map[string]string, resolver resolver.Resolver) error {
	operatorName := newOv.Spec.Operator.Name

	instance, err := kc.GetInstance(instanceName, namespace)
	if err != nil {
		return fmt.Errorf("failed to get instance: %v", err)
	}
	if instance == nil {
		return fmt.Errorf("instance %s/%s does not exist in the cluster", namespace, instanceName)
	}

	ov, err := kc.GetOperatorVersion(instance.Spec.OperatorVersion.Name, namespace)
	if err != nil {
		return fmt.Errorf("failed to retrieve existing operatorversion of instance %s/%s: %v", namespace, instanceName, err)
	}
	if ov == nil {
		return fmt.Errorf("no operatorversion for this operator installed yet for %s in namespace %s. Please use install command if you want to install new operator into cluster", operatorName, namespace)
	}
	oldVersion, err := semver.NewVersion(ov.Spec.Version)
	if err != nil {
		return fmt.Errorf("failed to parse %s as semver: %v", ov.Spec.Version, err)
	}
	newVersion, err := semver.NewVersion(newOv.Spec.Version)
	if err != nil {
		return fmt.Errorf("failed to parse %s as semver: %v", newOv.Spec.Version, err)
	}
	if !oldVersion.LessThan(newVersion) {
		return fmt.Errorf("upgraded version %s is the same or smaller as current version %s -> not upgrading", newOv.Spec.Version, ov.Spec.Version)
	}

	versionsInstalled, err := kc.OperatorVersionsInstalled(operatorName, namespace)
	if err != nil {
		return fmt.Errorf("failed to retrieve operatorversions: %v", err)
	}
	if !funk.ContainsString(versionsInstalled, newOv.Spec.Version) {
		if err := installDependencies(kc, newOv, instance, namespace, resolver); err != nil {
			return fmt.Errorf("failed to install dependencies of operatorversion %s/%s: %v", namespace, newOv.Name, err)
		}

		if _, err := kc.InstallOperatorVersionObjToCluster(newOv, namespace); err != nil {
			return fmt.Errorf("failed to update operatorversion %s/%s to version %s: %v", namespace, newOv.Name, newOv.Spec.Version, err)
		}
		clog.Printf("operatorversion %s/%s created", namespace, newOv.Name)
	}

	if err = kc.UpdateInstance(instanceName, namespace, convert.StringPtr(newOv.Name), parameters, nil, false, 0); err != nil {
		return fmt.Errorf("failed to update instance for new operatorversion %s/%s", namespace, newOv.Name)
	}
	clog.Printf("instance %s/%s updated", namespace, instanceName)

	return nil
}

func installDependencies(
	kc *kudo.Client,
	ov *v1beta1.OperatorVersion,
	instance *v1beta1.Instance,
	namespace string,
	resolver resolver.Resolver) error {
	resources := packages.Resources{
		Operator:        nil,
		OperatorVersion: ov,
		Instance:        instance,
	}

	dependencies, err := dependencies.Resolve(resources, resolver)
	if err != nil {
		return err
	}

	for _, dependency := range dependencies {
		installed, err := kc.OperatorVersionsInstalled(dependency.OperatorVersion.Name, namespace)
		if err != nil {
			return fmt.Errorf(
				"failed to retrieve operatorversion of dependency %s/%s: %v",
				namespace,
				dependency.OperatorVersion.Name,
				err)
		}

		if !funk.ContainsString(installed, dependency.OperatorVersion.Spec.Version) {
			dependency.Operator.SetNamespace(namespace)
			dependency.OperatorVersion.SetNamespace(namespace)

			if err := install.OperatorAndOperatorVersion(kc, dependency.Operator, dependency.OperatorVersion); err != nil {
				return err
			}
		}
	}

	return nil
}
