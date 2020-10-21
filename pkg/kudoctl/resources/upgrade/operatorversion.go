package upgrade

import (
	"fmt"

	"github.com/Masterminds/semver/v3"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	deps "github.com/kudobuilder/kudo/pkg/kudoctl/resources/dependencies"
	"github.com/kudobuilder/kudo/pkg/kudoctl/resources/install"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
	"github.com/kudobuilder/kudo/pkg/util/convert"
)

// OperatorVersion upgrades an OperatorVersion and its Instance.
// For the updated Instance, new parameters can be provided.
func OperatorVersion(
	kc *kudo.Client,
	newOv *kudoapi.OperatorVersion,
	instanceName string,
	parameters map[string]string,
	dependencies []deps.Dependency) error {

	ov, err := operatorVersionFromInstance(kc, instanceName, newOv.Namespace)
	if err != nil {
		return err
	}

	if err := compareVersions(ov.Spec.Version, newOv.Spec.Version); err != nil {
		return err
	}

	o, err := kc.GetOperator(newOv.Spec.Operator.Name, newOv.Namespace)
	if err != nil {
		return fmt.Errorf("failed to retrieve operator %s/%s: %v", newOv.Namespace, newOv.Spec.Operator.Name, err)
	}

	if err := install.OperatorAndOperatorVersion(kc, o, newOv, dependencies); err != nil {
		return fmt.Errorf("failed to install new operatorversion %s/%s: %v", newOv.Namespace, newOv.Name, err)
	}

	if err = kc.UpdateInstance(instanceName, newOv.Namespace, convert.StringPtr(newOv.Name), parameters, nil, false, 0); err != nil {
		return fmt.Errorf("failed to update instance for new operatorversion %s/%s: %v", newOv.Namespace, newOv.Name, err)
	}

	clog.Printf("instance %s/%s updated", newOv.Namespace, instanceName)
	return nil
}

func operatorVersionFromInstance(
	kc *kudo.Client,
	instanceName string,
	namespace string) (*kudoapi.OperatorVersion, error) {
	instance, err := kc.GetInstance(instanceName, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %v", err)
	}

	if instance == nil {
		return nil, fmt.Errorf("instance %s/%s does not exist in the cluster", namespace, instanceName)
	}

	operatorVersionName := instance.Spec.OperatorVersion.Name

	operatorVersion, err := kc.GetOperatorVersion(operatorVersionName, namespace)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to retrieve existing operatorversion of instance %s/%s: %v", namespace, instanceName, err)
	}

	if operatorVersion == nil {
		return nil, fmt.Errorf("operatorversion %s/%s does not exist in the cluster", namespace, operatorVersionName)
	}

	return operatorVersion, nil
}

func compareVersions(old string, new string) error {
	oldVersion, err := semver.NewVersion(old)
	if err != nil {
		return fmt.Errorf("failed to parse %s as semver: %v", old, err)
	}

	newVersion, err := semver.NewVersion(new)
	if err != nil {
		return fmt.Errorf("failed to parse %s as semver: %v", new, err)
	}

	if !oldVersion.LessThan(newVersion) {
		return fmt.Errorf("upgraded version %s is the same or smaller as current version %s -> not upgrading", new, old)
	}

	return nil
}
