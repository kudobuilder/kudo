package resources

import (
	"fmt"
	"strings"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/kudo"
)

// Install installs package CRDs.
// If skipInstance is set to true, only a package's Operator and OperatorVersion is installed.
func Install(kc *kudo.Client, resources *packages.PackageCRDs, skipInstance bool, instanceName, namespace string, parameters map[string]string) error {
	// PRE-INSTALLATION SETUP
	operatorName := resources.Operator.ObjectMeta.Name
	clog.V(3).Printf("operator name: %v", operatorName)
	operatorVersion := resources.OperatorVersion.Spec.Version
	clog.V(3).Printf("operator version: %v", operatorVersion)

	// make sure that our instance object is up to date with overrides from commandline
	applyInstanceOverrides(resources.Instance, instanceName, parameters)
	// this validation cannot be done earlier because we need to do it after applying things from commandline
	if err := validate(resources, skipInstance); err != nil {
		return err
	}

	if err := kc.ValidateServerForOperator(resources.Operator); err != nil {
		return err
	}

	if !kc.OperatorExistsInCluster(resources.Operator.ObjectMeta.Name, namespace) {
		if _, err := kc.InstallOperatorObjToCluster(resources.Operator, namespace); err != nil {
			return fmt.Errorf("failed to install %s-operator.yaml: %w", operatorName, err)
		}
		clog.Printf("operator.%s/%s created", resources.Operator.APIVersion, resources.Operator.Name)
	}

	versionsInstalled, err := kc.OperatorVersionsInstalled(operatorName, namespace)
	if err != nil {
		return fmt.Errorf("failed to retrieve existing operator versions: %w", err)
	}
	if !versionExists(operatorVersion, versionsInstalled) {
		if _, err := kc.InstallOperatorVersionObjToCluster(resources.OperatorVersion, namespace); err != nil {
			return fmt.Errorf("failed to install %s-operatorversion.yaml: %w", operatorName, err)
		}
		clog.Printf("operatorversion.%s/%s created", resources.OperatorVersion.APIVersion, resources.OperatorVersion.Name)
	}

	if skipInstance {
		return nil
	}

	instanceName = resources.Instance.ObjectMeta.Name
	instanceExists, err := kc.InstanceExistsInCluster(operatorName, namespace, resources.OperatorVersion.Spec.Version, instanceName)
	if err != nil {
		return fmt.Errorf("failed to verify existing instance: %w", err)
	}

	if !instanceExists {
		if _, err := kc.InstallInstanceObjToCluster(resources.Instance, namespace); err != nil {
			return fmt.Errorf("failed to install instance %s: %w", instanceName, err)
		}
		clog.Printf("instance.%s/%s created", resources.Instance.APIVersion, resources.Instance.Name)
	} else {
		return clog.Errorf("cannot install instance '%s' of operator '%s-%s' because an instance of that name already exists in namespace %s",
			instanceName, operatorName, resources.OperatorVersion.Spec.Version, namespace)
	}
	return nil
}

func applyInstanceOverrides(instance *v1alpha1.Instance, instanceName string, parameters map[string]string) {
	if instanceName != "" {
		instance.ObjectMeta.SetName(instanceName)
		clog.V(3).Printf("instance name: %v", instanceName)
	}
	if parameters != nil {
		instance.Spec.Parameters = parameters
		clog.V(3).Printf("parameters in use: %v", parameters)
	}
}

func validate(resources *packages.PackageCRDs, skipInstance bool) error {
	if skipInstance {
		// right now we are just validating parameters on instance, if we're not creating instance right now, there is nothing to validate
		clog.V(3).Printf("skipping instance...")
		return nil
	}
	parameters := resources.OperatorVersion.Spec.Parameters
	missingParameters := []string{}
	for _, p := range parameters {
		if p.Required && p.Default == nil {
			_, ok := resources.Instance.Spec.Parameters[p.Name]
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
