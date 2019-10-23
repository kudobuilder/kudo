package crds

import (
	"fmt"
	"strings"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

// Install installs package CRDs.
// If skipInstance is set to true, only a package's Operator and OperatorVersion is installed.
func Install(kc *kudo.Client, crds *packages.PackageCRDs, namespace string, skipInstance bool) error {
	// PRE-INSTALLATION SETUP
	operatorName := crds.Operator.ObjectMeta.Name
	clog.V(3).Printf("operator name: %v", operatorName)
	operatorVersion := crds.OperatorVersion.Spec.Version
	clog.V(3).Printf("operator version: %v", operatorVersion)

	if err := validate(crds, skipInstance); err != nil {
		return err
	}

	if err := kc.ValidateServerForOperator(crds.Operator); err != nil {
		return err
	}

	if !kc.OperatorExistsInCluster(crds.Operator.ObjectMeta.Name, namespace) {
		if _, err := kc.InstallOperatorObjToCluster(crds.Operator, namespace); err != nil {
			return fmt.Errorf("failed to install %s-operator.yaml: %w", operatorName, err)
		}
		clog.Printf("operator.%s/%s created", crds.Operator.APIVersion, crds.Operator.Name)
	}

	versionsInstalled, err := kc.OperatorVersionsInstalled(operatorName, namespace)
	if err != nil {
		return fmt.Errorf("failed to retrieve existing operator versions: %w", err)
	}
	if !versionExists(operatorVersion, versionsInstalled) {
		if _, err := kc.InstallOperatorVersionObjToCluster(crds.OperatorVersion, namespace); err != nil {
			return fmt.Errorf("failed to install %s-operatorversion.yaml: %w", operatorName, err)
		}
		clog.Printf("operatorversion.%s/%s created", crds.OperatorVersion.APIVersion, crds.OperatorVersion.Name)
	}

	if skipInstance {
		return nil
	}

	instanceName := crds.Instance.ObjectMeta.Name
	instanceExists, err := kc.InstanceExistsInCluster(operatorName, namespace, crds.OperatorVersion.Spec.Version, instanceName)
	if err != nil {
		return fmt.Errorf("failed to verify existing instance: %w", err)
	}

	if !instanceExists {
		if _, err := kc.InstallInstanceObjToCluster(crds.Instance, namespace); err != nil {
			return fmt.Errorf("failed to install instance %s: %w", instanceName, err)
		}
		clog.Printf("instance.%s/%s created", crds.Instance.APIVersion, crds.Instance.Name)
	} else {
		return clog.Errorf("cannot install instance '%s' of operator '%s-%s' because an instance of that name already exists in namespace %s",
			instanceName, operatorName, crds.OperatorVersion.Spec.Version, namespace)
	}
	return nil
}

func validate(crds *packages.PackageCRDs, skipInstance bool) error {
	if skipInstance {
		// right now we are just validating parameters on instance, if we're not creating instance right now, there is nothing to validate
		clog.V(3).Printf("skipping instance...")
		return nil
	}
	parameters := crds.OperatorVersion.Spec.Parameters
	missingParameters := []string{}
	for _, p := range parameters {
		if p.Required && p.Default == nil {
			_, ok := crds.Instance.Spec.Parameters[p.Name]
			if !ok {
				missingParameters = append(missingParameters, p.Name)
			}
		}
	}

	if len(missingParameters) > 0 {
		return clog.Errorf("missing required parameters during installation: %s", strings.Join(missingParameters, ","))
	}
	return nil
}

func versionExists(version string, versions []string) bool {
	for _, v := range versions {
		if v == version {
			return true
		}
	}

	return false
}
