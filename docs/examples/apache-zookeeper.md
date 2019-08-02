---
title: Apache Zookeeper
type: docs
---

# Apache Zookeeper

Create a Zookeeper cluster
```bash
$ kubectl kudo install zookeeper --instance=zk
operator.kudo.dev/v1alpha1/zookeeper created
operatorversion.kudo.dev/v1alpha1/zookeeper-0.1.0 created
instance.kudo.dev/v1alpha1/zk created
```

`kudo install zookeeper` creates the `Operator`, `OperatorVersion` and `Instance` of the Zookeeper package.
When an instance is created, the default `deploy` plan is executed

```
$ kubectl get planexecutions
NAME                  AGE
zk-deploy-392770000   11s
```

The statefulset defined in the `OperatorVersion` comes up with 3 pods:

```bash
$ kubectl get statefulset zk-zookeeper
NAME           READY   AGE
zk-zookeeper   3/3     1m20s
```

```bash
$ kubectl get pods
NAME             READY   STATUS    RESTARTS   AGE
zk-zookeeper-0   1/1     Running   0          3m43s
zk-zookeeper-1   1/1     Running   0          3m43s
zk-zookeeper-2   1/1     Running   0          3m43s
```
