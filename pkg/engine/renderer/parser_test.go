package renderer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseKubernetesObjects_UnknownType(t *testing.T) {
	objects, err := YamlToObject(`apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    app: prometheus-operator
    release: prometheus-kubeaddons
  name: spark-cluster-monitor
  annotations:
    foo: |-
      ---
      multiline
spec:
  endpoints:
    - interval: 5s
      port: metrics
  selector:
    matchLabels:
      spark/servicemonitor: true`)

	if err != nil {
		t.Errorf("Expecting no error but got %s", err)
	}
	assert.Equal(t, 1, len(objects))
}

func TestParseKubernetesObjects_KnownType(t *testing.T) {
	obj, err := YamlToObject(`apiVersion: apps/v1
kind: Deployment
metadata:
name: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
     app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.7.9
        ports:
        - containerPort: 80
        env:
        - name: PARAM_ENV
          value: 1`)

	if err != nil {
		t.Errorf("Expecting no error but got %s", err)
	}

	assert.Equal(t, "Deployment", obj[0].GetObjectKind().GroupVersionKind().Kind)
}

func TestParseKubernetesObjects_EmptyListOfObjects(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{"empty", ""},
		{"empty line", `
`},
		{"empty lines", `

`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			objects, err := YamlToObject(test.yaml)
			assert.NoError(t, err)
			assert.Empty(t, objects)
		})
	}
}
