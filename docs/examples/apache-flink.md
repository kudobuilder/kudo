---
title: Apache Flink
type: docs
---

# Apache Flink

https://ci.apache.org/projects/flink/flink-docs-stable/ops/deployment/kubernetes.html

There's a note about running `minikube ssh 'sudo ip link set docker0 promisc on'`, need to dive into this and see if we understand the requirement.


## Flink Demo

This demo follows the outline provided by the [DC/OS](https://github.com/dcos/demos/tree/master/flink-k8s/1.11) demo

### Architecture

We should modify the demo image to have everything run on K8s

## Prerequisites

Install the Kafka Operator and OperatorVersion

```bash
kubectl apply -f config/samples/kafka-operator.yaml
kubectl apply -f config/samples/kafka-operatorversion.yaml
```

Install the Flink Operator and OperatorVersion
```bash
kubectl apply -f config/samples/flink.yaml
```


## Deploy Plan
The deployplan has several stages that can be read from the Flink file, but we ouline the Phases and steps here:

### Kafka Phase

#### Kafka Step
The first phase is to create and deploy a Kafka instance:

#### Create Topic Step

The second Step in the phase runs a `Job` that creates a topic

### Flink Phase

Now we need to install Flink on Kubernetes


### Flink Generator Phase

This phase deplos a `Flink` Genenerator that mirrors the deployment [here](https://github.com/dcos/demos/blob/master/flink-k8s/1.11/generator/flink-demo-generator.yaml)


### Upload Flink Job to Flink

We need to figure this out, but it could be a simple image that

* downloads file from a URL that's an environment variable
* uses CURL to upload the file into Flink

### Flink Actor Phase

This phase deploys a `Flink` actor that mirrors the deployment [here](kubectl apply -f https://raw.githubusercontent.com/dcos/demos/master/flink-k8s/1.11/actor/flink-demo-actor.yaml)



# Scratch
```bash
cat <<EOF | kubectl apply -f -
apiVersion: kudo.dev/v1alpha1
kind: PlanExecution
metadata:
  name: upload
  namespace: default
spec:
  instance:
    kind: Instance
    name: demo
    namespace: default
  planName: upload
EOF
```
