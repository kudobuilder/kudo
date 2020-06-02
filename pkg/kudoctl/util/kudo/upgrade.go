package kudo

import (
	"fmt"

	"github.com/Masterminds/semver"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/util/convert"
)

// UpgradeOperatorVersion upgrades an OperatorVersion and its Instance.
// For the updated Instance, new parameters can be provided.
func UpgradeOperatorVersion(kc *Client, newOv *v1beta1.OperatorVersion, instanceName, namespace string, parameters map[string]string) error {
	operatorName := newOv.Spec.Operator.Name

	instance, err := kc.GetInstance(instanceName, namespace)
	if err != nil {
		return fmt.Errorf("failed to get instance: %v", err)
	}
	if instance == nil {
		return fmt.Errorf("instance %s in namespace %s does not exist in the cluster", instanceName, namespace)
	}

	ov, err := kc.GetOperatorVersion(instance.Spec.OperatorVersion.Name, namespace)
	if err != nil {
		return fmt.Errorf("failed to retrieve existing operator version: %v", err)
	}
	if ov == nil {
		return fmt.Errorf("no operator version for this operator installed yet for %s in namespace %s. Please use install command if you want to install new operator into cluster", operatorName, namespace)
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
		return fmt.Errorf("failed to retrieve operator versions: %v", err)
	}
	if !versionExists(newOv.Spec.Version, versionsInstalled) {
		if _, err := kc.InstallOperatorVersionObjToCluster(newOv, namespace); err != nil {
			return fmt.Errorf("failed to update %s-operatorversion.yaml to version %s: %v", operatorName, newOv.Spec.Version, err)
		}
		clog.Printf("operatorversion.%s/%s created", newOv.APIVersion, newOv.Name)
	}

	if err = kc.UpdateInstance(instanceName, namespace, convert.StringPtr(newOv.Name), parameters, nil, false, 0); err != nil {
		return fmt.Errorf("failed to update instance for new OperatorVersion %s", newOv.Name)
	}
	clog.Printf("instance.%s/%s updated", instance.APIVersion, instanceName)

	return nil
}
