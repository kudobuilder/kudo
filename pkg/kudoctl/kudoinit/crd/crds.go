//Defines the CRDs that the KUDO manager implements and watches.
package crd

import (
	"context"
	"fmt"
	"os"

	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

// Ensure kudoinit.Step is implemented
var _ kudoinit.Step = &Initializer{}

// Initializer represents custom resource definitions needed to run KUDO
type Initializer struct {
	Operator        *apiextv1beta1.CustomResourceDefinition
	OperatorVersion *apiextv1beta1.CustomResourceDefinition
	Instance        *apiextv1beta1.CustomResourceDefinition
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

func (c Initializer) PreInstallVerify(client *kube.Client, result *verifier.Result) error {
	return nil
}

func (c Initializer) VerifyInstallation(client *kube.Client, result *verifier.Result) error {
	if err := c.verifyInstallation(client.ExtClient.ApiextensionsV1beta1(), c.Operator, result); err != nil {
		return err
	}
	if err := c.verifyInstallation(client.ExtClient.ApiextensionsV1beta1(), c.OperatorVersion, result); err != nil {
		return err
	}
	if err := c.verifyInstallation(client.ExtClient.ApiextensionsV1beta1(), c.Instance, result); err != nil {
		return err
	}
	return nil
}

// Install uses Kubernetes client to install KUDO Crds.
func (c Initializer) Install(client *kube.Client) error {
	if err := c.install(client.ExtClient.ApiextensionsV1beta1(), c.Operator); err != nil {
		return err
	}
	if err := c.install(client.ExtClient.ApiextensionsV1beta1(), c.OperatorVersion); err != nil {
		return err
	}
	if err := c.install(client.ExtClient.ApiextensionsV1beta1(), c.Instance); err != nil {
		return err
	}
	return nil
}

func (c Initializer) verifyInstallation(client v1beta1.CustomResourceDefinitionsGetter, crd *apiextv1beta1.CustomResourceDefinition, result *verifier.Result) error {
	existingCrd, err := client.CustomResourceDefinitions().Get(context.TODO(), crd.Name, v1.GetOptions{})
	if err != nil {
		if os.IsTimeout(err) {
			return err
		}
		if kerrors.IsNotFound(err) {
			result.AddErrors(fmt.Sprintf("CRD %s is not installed", crd.Name))
			return nil
		}
		return fmt.Errorf("failed to retrieve CRD %s: %v", crd.Name, err)
	}
	if existingCrd.Spec.Version != crd.Spec.Version {
		result.AddErrors(fmt.Sprintf("Installed CRD %s has invalid version %s, expected %s", crd.Name, existingCrd.Spec.Version, crd.Spec.Version))
		return nil
	}
	return nil
}

func (c Initializer) install(client v1beta1.CustomResourceDefinitionsGetter, crd *apiextv1beta1.CustomResourceDefinition) error {
	_, err := client.CustomResourceDefinitions().Create(context.TODO(), crd, v1.CreateOptions{})
	if kerrors.IsAlreadyExists(err) {
		clog.V(4).Printf("crd %v already exists", crd.Name)
		return nil
	}
	return err
}

func embeddedCRD(path string) *apiextv1beta1.CustomResourceDefinition {
	operatorYaml := MustAsset(path)
	crd := &apiextv1beta1.CustomResourceDefinition{}
	err := yaml.UnmarshalStrict(operatorYaml, crd)
	if err != nil {
		panic(fmt.Sprintf("cannot unmarshal embedded content of %s: %v", path, err))
	}
	return crd
}
