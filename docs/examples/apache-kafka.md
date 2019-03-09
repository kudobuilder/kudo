---
title: Apache Kafka
type: docs
---

# Apache Kafka

## Dependencies

Kafka depends on Zookeeper so we need to run it first. Follow the [Zookeeper example](/docs/examples/apache-zookeeper/) to run a basic cluster. 

## Run Kafka

Create a `Framework` object for Kafka
```bash
$ kubectl apply -f config/samples/kafka-framework.yaml
framework.kudo.k8s.io "kafka" created
```

Create a `FrameworkVersion` for the Kafka  `Framework`

```bash
$ kubectl apply -f config/samples/kafka-frameworkversion.yaml
frameworkversion.kudo.k8s.io "kafka-2.11-2.4.0" created
```
 

Create an Instance of Kafka
```
$ kubectl apply -f config/samples/kafka-instance.yaml
instance.kudo.k8s.io "kafka" created
```

When an instance is created, the default `deploy` plan is executed

```
$ kubectl get planexecutions
NAME                    AGE
kafka-deploy-91712000   13s
zk-deploy-392770000     3m
```

The statefulset defined in the `FrameworkVersion` comes up with 1 pod:

```bash
$ kubectl get statefulset kafka-kafka
NAME          DESIRED   CURRENT   AGE
kafka-kafka   1         1         56s
```

```bash
$ kubectl get pods
NAME            READY     STATUS             RESTARTS   AGE
kafka-kafka-0   1/1       Running            0          1m
zk-zk-0         1/1       Running            0          4m
zk-zk-1         1/1       Running            0          4m
zk-zk-2         1/1       Running            0          4m
```