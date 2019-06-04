package helm

import (
	"fmt"
	"testing"

	"encoding/json"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/onsi/gomega"
)

func TestHelmImportMeta(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	base := "../../../test/charts/mysql"

	framework, err := loadMetadata(base)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(framework.Name).To(gomega.Equal("mysql"))
	g.Expect(framework.Spec.Description).To(gomega.Equal("Fast, reliable, scalable, and easy to use open-source relational database system."))
	if g.Expect(len(framework.Spec.Maintainers)).To(gomega.Equal(2)) {
		g.Expect(framework.Spec.Maintainers[0]).To(gomega.Equal(kudo.Maintainer("olemarkus <o.with@sportradar.com>")))
		g.Expect(framework.Spec.Maintainers[1]).To(gomega.Equal(kudo.Maintainer("viglesiasce <viglesias@google.com>")))
	}
}

func TestHelmLoadTemplates(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	base := "../../../test/charts/mysql"

	templates, err := loadTemplates(base)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	g.Expect(templates["configurationFiles-configmap.yaml"]).NotTo(gomega.BeEmpty())
	g.Expect(templates["deployment.yaml"]).NotTo(gomega.BeEmpty())
	g.Expect(templates["initializationFiles-configmap.yaml"]).NotTo(gomega.BeEmpty())
	g.Expect(templates["pvc.yaml"]).NotTo(gomega.BeEmpty())
	g.Expect(templates["secrets.yaml"]).NotTo(gomega.BeEmpty())
	g.Expect(templates["servicemonitor.yaml"]).NotTo(gomega.BeEmpty())
	g.Expect(templates["svc.yaml"]).NotTo(gomega.BeEmpty())
	g.Expect(templates["NOTES.txt"]).To(gomega.BeEmpty())
	g.Expect(templates["_helpers.tpl"]).To(gomega.BeEmpty())
}

func TestLoadParamaters(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	// This package is expected to be run from the top level
	base := "../../../test/charts/mysql"

	params, err := loadParameters(base)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	b, _ := json.MarshalIndent(params, "", "\t")
	fmt.Printf("%v", string(b))
}

func TestGetFrameworkFromHelm(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	base := "../../../test/charts/mysql"

	f, fv, err := Import(base)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	b, _ := json.MarshalIndent(f, "framework", "\t")
	fmt.Printf("%v", string(b))
	fmt.Printf("\n\n\n\n\n\n")
	b, _ = json.MarshalIndent(fv, "frameworkversion", "\t")
	fmt.Printf("%v", string(b))
}
