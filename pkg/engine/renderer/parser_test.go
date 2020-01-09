package renderer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseKubernetesObjects_UnknownType(t *testing.T) {
	_, err := YamlToObject(`apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    app: prometheus-operator
    release: prometheus-kubeaddons
  name: spark-cluster-monitor
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
