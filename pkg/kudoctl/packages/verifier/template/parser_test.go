package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"

	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
)

func TestTemplate_Parameters(t *testing.T) {
	tplate := `apiVersion: policy/v1beta1
kind: PodDisruptionBudget
metadata:
  name: {{ .Name }}-pdb
  namespace: {{ .Namespace }}
  labels:
    app: zookeeper
    zookeeper: {{ .Params.Foo }}
spec:
  selector:
    matchLabels:
      app: zookeeper
      zookeeper: {{ .Name }}
  maxUnavailable: 1
    {{ if .Params.JVM_OPT_AVAILABLE_PROCESSORS }}
	processors={{ .Params.JVM_OPT_AVAILABLE_PROCESSORS }}
    {{ end }}
	{{ if eq .Params.AUTHORIZATION_ENABLED "true" }}
	foo is authorized
    {{ end }}
	{{ if .Params.CUSTOM_CASSANDRA_YAML_BASE64 }}
    {{ .Params.CUSTOM_CASSANDRA_YAML_BASE64 | b64dec }}
    {{ end }}
	{{ .Xyx.Foo }}
`
	// {{ .Xyz.Foo }} added to tests to allow for additional extensions to templating which will be ignored by linter until they are add to the set of lint validators

	var templates = packages.Templates{}
	templates["example.yaml"] = tplate

	tnodes := getNodeMap(templates)
	nodes := tnodes["example.yaml"]

	assert.Equal(t, 4, len(nodes.parameters))
	params := []string{"Foo", "JVM_OPT_AVAILABLE_PROCESSORS", "AUTHORIZATION_ENABLED", "CUSTOM_CASSANDRA_YAML_BASE64"}
	for _, param := range params {
		if !funk.ContainsString(nodes.parameters, param) {
			t.Fatalf("missing %q parameter", param)
		}
	}
	assert.Equal(t, 2, len(nodes.implicitParams))
	implicits := []string{"Name", "Namespace"}
	for _, param := range implicits {
		if !funk.ContainsString(nodes.implicitParams, param) {
			t.Fatalf("missing %q implicit parameter", param)
		}
	}

}

func TestBadTemplate(t *testing.T) {
	tplate := `apiVersion: policy/v1beta1
kind: PodDisruptionBudget
metadata:
  name: {{ .Name }}-pdb
  namespace: {{ .Namespace }}
  {{ end }}

`
	var templates = packages.Templates{}
	templates["example.yaml"] = tplate

	tnodes := getNodeMap(templates)
	nodes := tnodes["example.yaml"]

	assert.Equal(t, 0, len(nodes.parameters))
	assert.Equal(t, 0, len(nodes.implicitParams))

	assert.Equal(t, `template file "example.yaml" reports the following error: template: example.yaml:6: unexpected {{end}}`, *nodes.error)
}
