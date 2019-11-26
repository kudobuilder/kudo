package packages

import (
	"testing"

	"gotest.tools/assert"
)

func TestTemplate_Parameters(t *testing.T) {
	template := `apiVersion: policy/v1beta1
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

`
	var templates = Templates{}
	templates["example.yaml"] = template

	tnodes := templates.Nodes()
	nodes := tnodes["example.yaml"]

	assert.Equal(t, 4, len(nodes.Parameters))
	params := []string{"Foo", "JVM_OPT_AVAILABLE_PROCESSORS", "AUTHORIZATION_ENABLED", "CUSTOM_CASSANDRA_YAML_BASE64"}
	for _, param := range params {
		if !contains(nodes.Parameters, param) {
			t.Fatalf("missing %q parameter", param)
		}
	}
	assert.Equal(t, 2, len(nodes.ImplicitParams))
	implicits := []string{"Name", "Namespace"}
	for _, param := range implicits {
		if !contains(nodes.ImplicitParams, param) {
			t.Fatalf("missing %q implicit parameter", param)
		}
	}

}

func TestBadTemplate(t *testing.T) {
	template := `apiVersion: policy/v1beta1
kind: PodDisruptionBudget
metadata:
  name: {{ .Name }}-pdb
  namespace: {{ .Namespace }}
  {{ end }}

`
	var templates = Templates{}
	templates["example.yaml"] = template

	tnodes := templates.Nodes()
	nodes := tnodes["example.yaml"]

	assert.Equal(t, 0, len(nodes.Parameters))
	assert.Equal(t, 0, len(nodes.ImplicitParams))

	assert.Equal(t, `template file "example.yaml" reports the following error: template: example.yaml:6: unexpected {{end}}`, *nodes.Error)
}

func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}
