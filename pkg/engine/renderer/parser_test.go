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

	assert.NoError(t, err)
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

	assert.NoError(t, err)
	assert.Equal(t, "Deployment", obj[0].GetObjectKind().GroupVersionKind().Kind)
}

func TestParseKubernetesObjects_MoreThanOne(t *testing.T) {
	objects, err := YamlToObject(`apiVersion: foo
kind: Foo
metadata:
  name: foo1
---
apiVersion: foo
kind: Foo
metadata:
name: foo2`)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(objects))
}
