package template

import "testing"

func TestParseKubernetesObjects_UnknownType(t *testing.T) {
	_, err := ParseKubernetesObjects(`apiVersion: monitoring.coreos.com/v1
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
