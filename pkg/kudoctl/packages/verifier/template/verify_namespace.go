package template

import (
	"fmt"
	"regexp"

	"github.com/kudobuilder/kudo/pkg/engine/renderer"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/verifier"
)

var _ packages.Verifier = &NamespaceVerifier{}

type NamespaceVerifier struct{}

func (n NamespaceVerifier) Verify(pf *packages.Files) verifier.Result {
	res := verifier.NewResult()

	n.namespaceKindNotUsed(pf, &res)
	n.checkNamespaceManifest(pf, &res)

	return res
}

func (n NamespaceVerifier) checkNamespaceManifest(pf *packages.Files, res *verifier.Result) {
	if pf.Operator.NamespaceManifest != "" {
		val, ok := pf.Templates[pf.Operator.NamespaceManifest]
		if !ok {
			res.AddErrors(fmt.Sprintf("NamespaceManifest %q not found in /templates folder", pf.Operator.NamespaceManifest))
			return
		}
		ns, err := renderer.YamlToObject(val)
		if err != nil {
			res.AddErrors(fmt.Sprintf("Unable to marshal NamespaceManifest %q ", pf.Operator.NamespaceManifest))
			return
		}
		switch count := len(ns); {
		case count == 0:
			res.AddErrors(fmt.Sprintf("NamespaceManifest %q found but does not contain a manifest", pf.Operator.NamespaceManifest))
		case count == 1:
			if ns[0].GetObjectKind().GroupVersionKind().Kind != "Namespace" {
				res.AddErrors(fmt.Sprintf("NamespaceManifest %q found but manifest is not kind: Namespace", pf.Operator.NamespaceManifest))
			}
		case count > 1:
			res.AddErrors(fmt.Sprintf("NamespaceManifest %q found but contains %v manifests which is greater than 1", pf.Operator.NamespaceManifest, count))
		}
	}
}

func (n NamespaceVerifier) namespaceKindNotUsed(pf *packages.Files, res *verifier.Result) {
	// it would be great if we could marshal the templates into an unstructured runtime.Object to confirm Namespace
	// however with templating we don't have the ability to have all required information to do so.
	// best effort is to confirm that "kind: Namespace" is not used.
	// kind: + (0 to many <spaces>) + Namespace
	nsRegex := regexp.MustCompile(`kind:\s*Namespace`)
	// scan through all files for `kind: Namespace`
	for name, file := range pf.Templates {
		// ignore the namespace manifest
		if name == pf.Operator.NamespaceManifest {
			continue
		}
		if nsRegex.MatchString(file) {
			res.AddErrors(fmt.Sprintf("template %q contains 'kind: Namespace' not allowed unless specified as 'NamespaceManifest'", name))
		}
	}
}
