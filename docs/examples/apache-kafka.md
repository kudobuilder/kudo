---
title: Apache Kafka
type: docs
---

# Apache Kafka

## Dependencies

Kafka depends on Zookeeper so we need to run it first. Follow the [Zookeeper example](/docs/examples/apache-zookeeper/) to run a basic cluster.

## Run Kafka

Create a Kafka cluster
```bash
$ kubectl kudo install kafka --instance=kafka
operator.kudo.dev/kafka unchanged
operatorversion.kudo.dev/v1alpha1/kafka-0.2.0 created
instance.kudo.dev/v1alpha1/kafka created
```

`kudo install kafka` creates the `Operator`, `OperatorVersion` and `Instance` CRDs of the Kafka package.
When an instance is created, the default `deploy` plan is executed

```
$ kubectl get planexecutions
NAME                    AGE
kafka-deploy-91712000   13s
zk-deploy-392770000     3m
```

The statefulset defined in the `OperatorVersion` comes up with 3 pods:

```bash
$ kubectl get statefulset kafka-kafka
NAME          READY   AGE
kafka-kafka   3/3     56s
```

```bash
$ kubectl get pods
NAME             READY   STATUS    RESTARTS   AGE
kafka-kafka-0    1/1     Running   0          83s
kafka-kafka-1    1/1     Running   0          61s
kafka-kafka-2    1/1     Running   0          34s
zk-zookeeper-0   1/1     Running   0          6m56s
zk-zookeeper-1   1/1     Running   0          6m56s
zk-zookeeper-2   1/1     Running   0          6m56s
```

You can find more details around configuring Kafka Cluster in [KUDO Kafka documentation](https://github.com/kudobuilder/operators/tree/master/repository/kafka)
