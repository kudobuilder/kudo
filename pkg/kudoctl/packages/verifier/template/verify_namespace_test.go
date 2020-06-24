package template

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

func TestNamespaceUsedInTemplate(t *testing.T) {

	templates := make(map[string]string)
	templates["foo.yaml"] = `
## this template contains a namespace
kind: Namespace

{{ Name }}
`
	pf := packages.Files{
		Templates: templates,
		Operator:  &packages.OperatorFile{},
	}
	verifier := NamespaceVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 0, len(res.Warnings))
	assert.Equal(t, 1, len(res.Errors))
	assert.Equal(t, "template \"foo.yaml\" contains 'kind: Namespace' not allowed unless specified as 'NamespaceManifest'", res.Errors[0])
}

func TestNamespaceNotUsedInTemplate(t *testing.T) {

	templates := make(map[string]string)
	templates["foo.yaml"] = `
## this template contains a namespace
kind: Name

{{ Name }}
`
	pf := packages.Files{
		Templates: templates,
		Operator:  &packages.OperatorFile{},
	}
	verifier := NamespaceVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 0, len(res.Warnings))
	assert.Equal(t, 0, len(res.Errors))
}

func TestNamespaceManifestFileMissing(t *testing.T) {

	templates := make(map[string]string)
	templates["foo.yaml"] = `
## this template contains a namespace
kind: Name

{{ Name }}
`
	pf := packages.Files{
		Templates: templates,
		Operator:  &packages.OperatorFile{NamespaceManifest: "bar.yaml"},
	}
	verifier := NamespaceVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 0, len(res.Warnings))
	assert.Equal(t, 1, len(res.Errors))
	assert.Equal(t, "NamespaceManifest \"bar.yaml\" not found in /templates folder", res.Errors[0])
}

func TestNamespaceManifestFileEmpty(t *testing.T) {

	templates := make(map[string]string)
	templates["foo.yaml"] = `
`
	pf := packages.Files{
		Templates: templates,
		Operator:  &packages.OperatorFile{NamespaceManifest: "foo.yaml"},
	}
	verifier := NamespaceVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 0, len(res.Warnings))
	assert.Equal(t, 1, len(res.Errors))
	assert.Equal(t, "NamespaceManifest \"foo.yaml\" found but does not contain a manifest", res.Errors[0])
}

func TestNamespaceManifestNotNamespace(t *testing.T) {

	templates := make(map[string]string)
	templates["foo.yaml"] = `apiVersion: v1
kind: Pod
metadata:
 labels:
    app: my-app`

	pf := packages.Files{
		Templates: templates,
		Operator:  &packages.OperatorFile{NamespaceManifest: "foo.yaml"},
	}
	verifier := NamespaceVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 0, len(res.Warnings))
	assert.Equal(t, 1, len(res.Errors))
	assert.Equal(t, "NamespaceManifest \"foo.yaml\" found but manifest is not kind: Namespace", res.Errors[0])
}

func TestNamespaceManifestMultiNamespace(t *testing.T) {

	templates := make(map[string]string)
	templates["foo.yaml"] = `apiVersion: foo
kind: Foo
metadata:
  name: foo1
---
apiVersion: foo
kind: Foo
metadata:
name: foo2`

	pf := packages.Files{
		Templates: templates,
		Operator:  &packages.OperatorFile{NamespaceManifest: "foo.yaml"},
	}
	verifier := NamespaceVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 0, len(res.Warnings))
	assert.Equal(t, 1, len(res.Errors))
	assert.Equal(t, "NamespaceManifest \"foo.yaml\" found but contains 2 manifests which is greater than 1", res.Errors[0])
}

func TestNamespaceManifestGood(t *testing.T) {

	templates := make(map[string]string)
	templates["foo.yaml"] = `apiVersion: v1
kind: Namespace
metadata:
 labels:
    app: my-app`

	pf := packages.Files{
		Templates: templates,
		Operator:  &packages.OperatorFile{NamespaceManifest: "foo.yaml"},
	}
	verifier := NamespaceVerifier{}
	res := verifier.Verify(&pf)

	assert.Equal(t, 0, len(res.Warnings))
	assert.Equal(t, 0, len(res.Errors))
}
