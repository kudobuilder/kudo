// Defines the CRDs that the KUDO manager implements and watches.
package crd

import (
	"context"
	"fmt"
	"os"
	"reflect"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	crdclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/kubernetes/status"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

// Ensure kudoinit.Step is implemented
var _ kudoinit.Step = &Initializer{}

// Initializer represents custom resource definitions needed to run KUDO
type Initializer struct {
	Operator        *apiextv1.CustomResourceDefinition
	OperatorVersion *apiextv1.CustomResourceDefinition
	Instance        *apiextv1.CustomResourceDefinition
}

// CRDs returns the runtime.Object representation of all the CRDs KUDO requires
func NewInitializer() Initializer {
	return Initializer{
		Operator:        embeddedCRD("config/crds/kudo.dev_operators.yaml"),
		OperatorVersion: embeddedCRD("config/crds/kudo.dev_operatorversions.yaml"),
		Instance:        embeddedCRD("config/crds/kudo.dev_instances.yaml"),
	}
}

func (c Initializer) String() string {
	return "crds"
}

// Resources returns all CRDs as array of runtime objects
func (c Initializer) Resources() []runtime.Object {
	return []runtime.Object{c.Operator, c.OperatorVersion, c.Instance}
}

// PreInstallVerify ensures that CRDs are not installed
func (c Initializer) PreInstallVerify(client *kube.Client, result *verifier.Result) error {
	if err := c.verifyIsNotInstalled(client.ExtClient.ApiextensionsV1(), c.Operator, result); (err != nil) || !result.IsValid() {
		return err
	}
	if err := c.verifyIsNotInstalled(client.ExtClient.ApiextensionsV1(), c.OperatorVersion, result); (err != nil) || !result.IsValid() {
		return err
	}
	if err := c.verifyIsNotInstalled(client.ExtClient.ApiextensionsV1(), c.Instance, result); (err != nil) || !result.IsValid() {
		return err
	}
	return nil
}

func (c Initializer) PreUpgradeVerify(client *kube.Client, result *verifier.Result) error {
	return nil
}

// VerifyInstallation ensures that all CRDs are installed and are the same as this CLI would install
func (c Initializer) VerifyInstallation(client *kube.Client, result *verifier.Result) error {
	apiClient := client.ExtClient.ApiextensionsV1()
	if err := c.verifyInstallation(apiClient, c.Operator, result); err != nil {
		return err
	}
	if err := c.verifyInstallation(apiClient, c.OperatorVersion, result); err != nil {
		return err
	}
	if err := c.verifyInstallation(apiClient, c.Instance, result); err != nil {
		return err
	}
	return nil
}

// VerifyServedVersion ensures that the api server provides the correct version of all CRDs that this client understands
func (c Initializer) VerifyServedVersion(client *kube.Client, expectedVersion string, result *verifier.Result) error {
	apiClient := client.ExtClient.ApiextensionsV1()
	if err := c.verifyServedVersion(apiClient, c.Operator.Name, expectedVersion, result); err != nil {
		return err
	}
	if err := c.verifyServedVersion(apiClient, c.OperatorVersion.Name, expectedVersion, result); err != nil {
		return err
	}
	if err := c.verifyServedVersion(apiClient, c.Instance.Name, expectedVersion, result); err != nil {
		return err
	}
	return nil
}

// Install uses Kubernetes client to install KUDO Crds.
func (c Initializer) Install(client *kube.Client) error {
	if err := c.apply(client.ExtClient.ApiextensionsV1(), c.Operator); err != nil {
		return err
	}
	if err := c.apply(client.ExtClient.ApiextensionsV1(), c.OperatorVersion); err != nil {
		return err
	}
	if err := c.apply(client.ExtClient.ApiextensionsV1(), c.Instance); err != nil {
		return err
	}
	return nil
}

// verifyIsNotInstalled is used to ensure that the cluster has no old KUDO version installed
func (c Initializer) verifyIsNotInstalled(client crdclient.CustomResourceDefinitionsGetter, crd *apiextv1.CustomResourceDefinition, result *verifier.Result) error {
	_, err := client.CustomResourceDefinitions().Get(context.TODO(), crd.Name, v1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	result.AddErrors(fmt.Sprintf("CRD %s is already installed. Did you mean to use --upgrade?", crd.Name))
	return nil
}

func (c Initializer) getCrdForVerify(client crdclient.CustomResourceDefinitionsGetter, crdName string, result *verifier.Result) (*apiextv1.CustomResourceDefinition, error) {
	existingCrd, err := client.CustomResourceDefinitions().Get(context.TODO(), crdName, v1.GetOptions{})
	if err != nil {
		if os.IsTimeout(err) {
			return nil, err
		}
		if kerrors.IsNotFound(err) {
			result.AddErrors(fmt.Sprintf("CRD %s is not installed", crdName))
			return nil, nil
		}
		return nil, fmt.Errorf("failed to retrieve CRD %s: %v", crdName, err)
	}
	return existingCrd, nil
}

// VerifyInstallation ensures that a single CRD is installed and is the same as this CLI would install
func (c Initializer) verifyInstallation(client crdclient.CustomResourceDefinitionsGetter, crd *apiextv1.CustomResourceDefinition, result *verifier.Result) error {
	existingCrd, err := c.getCrdForVerify(client, crd.Name, result)
	if err != nil || existingCrd == nil {
		return err
	}
	if !reflect.DeepEqual(existingCrd.Spec.Versions, crd.Spec.Versions) {
		result.AddErrors(fmt.Sprintf("Installed CRD versions do not match expected CRD versions (%v vs %v).", existingCrd.Spec.Versions, crd.Spec.Versions))
	}
	if healthy, msg, err := status.IsHealthy(existingCrd); !healthy || err != nil {
		if err != nil {
			return err
		}
		result.AddErrors(fmt.Sprintf("Installed CRD %s is not healthy: %v", crd.Name, msg))
		return nil
	}
	clog.V(2).Printf("CRD %s is installed with versions %v", crd.Name, existingCrd.Spec.Versions)
	return nil
}

// VerifyServedVersion ensures that the api server provides the correct version of a specific CRDs that this client understands
func (c Initializer) verifyServedVersion(client crdclient.CustomResourceDefinitionsGetter, crdName, version string, result *verifier.Result) error {
	existingCrd, err := c.getCrdForVerify(client, crdName, result)
	if err != nil || existingCrd == nil {
		return err
	}
	if healthy, msg, err := status.IsHealthy(existingCrd); !healthy || err != nil {
		if !healthy {
			result.AddErrors(msg)
		} else {
			result.AddErrors(err.Error())
		}
		return nil
	}

	var expectedVersion *apiextv1.CustomResourceDefinitionVersion
	var allNames = []string{}
	for _, v := range existingCrd.Spec.Versions {
		v := v
		allNames = append(allNames, v.Name)
		if v.Name == version {
			expectedVersion = &v
			break
		}
	}
	if expectedVersion == nil {
		result.AddErrors(fmt.Sprintf("Expected API version %s was not found for %s, api-server only supports %v. Please update your KUDO CLI.", version, crdName, allNames))
		return nil
	}
	if !expectedVersion.Served {
		result.AddErrors(fmt.Sprintf("Expected API version %s for %s is known to api-server, but is not served. Please update your KUDO CLI.", version, crdName))
	}
	return nil
}

func (c Initializer) apply(client crdclient.CustomResourceDefinitionsGetter, crd *apiextv1.CustomResourceDefinition) error {
	_, err := client.CustomResourceDefinitions().Create(context.TODO(), crd, v1.CreateOptions{})
	if kerrors.IsAlreadyExists(err) {
		// We need to be careful here and never delete/recreate CRDs, we would delete
		// all installed custom resources. We must have a correct update!
		clog.V(4).Printf("crd %v already exists, try to update", crd.Name)

		oldCrd, err := client.CustomResourceDefinitions().Get(context.TODO(), crd.Name, v1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get crd for update %s: %v", crd.Name, err)
		}

		// As we call update, we need to take over the resourceVersion
		crd.ResourceVersion = oldCrd.ResourceVersion
		_, err = client.CustomResourceDefinitions().Update(context.TODO(), crd, v1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update crd %s: %v", crd.Name, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to create crd %s: %v", crd.Name, err)
	}
	return nil
}

func embeddedCRD(path string) *apiextv1.CustomResourceDefinition {
	operatorYaml := MustAsset(path)
	crd := &apiextv1.CustomResourceDefinition{}
	err := yaml.UnmarshalStrict(operatorYaml, crd)
	if err != nil {
		panic(fmt.Sprintf("cannot unmarshal embedded content of %s: %v", path, err))
	}
	return crd
}
