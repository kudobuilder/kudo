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
`
	var templates Templates = Templates{}
	templates["example.yaml"] = template

	tnodes := templates.Nodes()
	nodes := tnodes["example.yaml"]

	assert.Equal(t, 1, len(nodes.Parameters()))
	assert.DeepEqual(t, []string{"Foo"}, nodes.Parameters())

	assert.Equal(t, 2, len(nodes.ImplicitParams()))
	implicits := []string{"Name", "Namespace"}
	assert.DeepEqual(t, implicits, nodes.ImplicitParams())
}
