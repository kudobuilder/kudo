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

	"github.com/kudobuilder/kudo/pkg/engine/health"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudoinit/prereq"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
	"github.com/kudobuilder/kudo/pkg/util/convert"
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
func NewInitializer(options kudoinit.Options) Initializer {
	var tinyCA *prereq.TinyCA
	if options.SelfSignedWebhookCA {
		tinyCA, _ = prereq.NewTinyCA(kudoinit.DefaultServiceName, options.Namespace)
	}

	return Initializer{
		Operator:        embeddedCRD("config/crds/kudo.dev_operators.yaml", options, tinyCA),
		OperatorVersion: embeddedCRD("config/crds/kudo.dev_operatorversions.yaml", options, tinyCA),
		Instance:        embeddedCRD("config/crds/kudo.dev_instances.yaml", options, tinyCA),
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
	if err := c.verifyIsNotInstalled(client.ExtClient.ApiextensionsV1beta1(), c.Operator, result); (err != nil) || !result.IsValid() {
		return err
	}
	if err := c.verifyIsNotInstalled(client.ExtClient.ApiextensionsV1beta1(), c.OperatorVersion, result); (err != nil) || !result.IsValid() {
		return err
	}
	if err := c.verifyIsNotInstalled(client.ExtClient.ApiextensionsV1beta1(), c.Instance, result); (err != nil) || !result.IsValid() {
		return err
	}
	return nil
}

func (c Initializer) PreUpgradeVerify(client *kube.Client, result *verifier.Result) error {
	return nil
}

// VerifyInstallation ensures that the CRDs are installed and have the correct and expected version
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
	if err := c.apply(client.ExtClient.ApiextensionsV1beta1(), c.Operator); err != nil {
		return err
	}
	if err := c.apply(client.ExtClient.ApiextensionsV1beta1(), c.OperatorVersion); err != nil {
		return err
	}
	if err := c.apply(client.ExtClient.ApiextensionsV1beta1(), c.Instance); err != nil {
		return err
	}
	return nil
}

func (c Initializer) verifyIsNotInstalled(client v1beta1.CustomResourceDefinitionsGetter, crd *apiextv1beta1.CustomResourceDefinition, result *verifier.Result) error {
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
	if err := health.IsHealthy(existingCrd); err != nil {
		result.AddErrors(fmt.Sprintf("Installed CRD %s is not healthy: %v", crd.Name, err))
		return nil
	}
	clog.V(2).Printf("CRD %s is installed with version %s", crd.Name, existingCrd.Spec.Versions[0].Name)
	return nil
}

func (c Initializer) apply(client v1beta1.CustomResourceDefinitionsGetter, crd *apiextv1beta1.CustomResourceDefinition) error {
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

func embeddedCRD(path string, options kudoinit.Options, tinyCA *prereq.TinyCA) *apiextv1beta1.CustomResourceDefinition {
	operatorYaml := MustAsset(path)
	crd := &apiextv1beta1.CustomResourceDefinition{}
	err := yaml.UnmarshalStrict(operatorYaml, crd)
	if err != nil {
		panic(fmt.Sprintf("cannot unmarshal embedded content of %s: %v", path, err))
	}

	crd.Spec.Conversion = &apiextv1beta1.CustomResourceConversion{
		Strategy:                 apiextv1beta1.WebhookConverter,
		ConversionReviewVersions: []string{"v1beta2", "v1beta1"},
		WebhookClientConfig: &apiextv1beta1.WebhookClientConfig{
			Service: &apiextv1beta1.ServiceReference{
				Name:      kudoinit.DefaultServiceName,
				Namespace: options.Namespace,
				Path:      convert.StringPtr("/admit-kudo-dev-v1beta1-instance"),
			},
		},
	}
	if tinyCA != nil {
		crd.Spec.Conversion.WebhookClientConfig.CABundle = tinyCA.CA.CertBytes()
	}

	return crd
}
