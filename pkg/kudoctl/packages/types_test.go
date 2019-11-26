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

`
	var templates Templates = Templates{}
	templates["example.yaml"] = template

	tnodes := templates.Nodes()
	nodes := tnodes["example.yaml"]

	assert.Equal(t, 2, len(nodes.Parameters))
	params := []string{"Foo", "JVM_OPT_AVAILABLE_PROCESSORS"}
	for _, param := range params {
		if !contains(nodes.Parameters, param) {
			t.Fatalf("missing %q parameter", param)
		}
	}
	assert.Equal(t, 2, len(nodes.ImplicitParams))
	implicits := []string{"Name", "Namespace"}
	for _, param := range implicits {
		if !contains(nodes.ImplicitParams, param) {
			t.Fatalf("missing %q parameter", param)
		}
	}

}

func contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}