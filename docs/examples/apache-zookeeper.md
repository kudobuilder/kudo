---
title: Apache Zookeeper
type: docs
---

# Apache Zookeeper

Create a `Operator` object for Zookeeper
```bash
$ kubectl apply -f config/samples/zookeeper-operator.yaml
operator.kudo.dev "zookeeper" created
```

Create a `OperatorVersion` for the Zookeeper  `Operator`

```bash
$ kubectl apply -f config/samples/zookeeper-operatorversion.yaml
operatorversion.kudo.dev "zookeeper-1.0" created
```


Create an Instance of Zookeeper
```
$ kubectl apply -f config/samples/zookeeper-instance.yaml
instance.kudo.dev "zk" created
```

When an instance is created, the default `deploy` plan is executed

```
$ kubectl get planexecutions
NAME                  AGE
zk-deploy-392770000   11s
```

The statefulset defined in the `OperatorVersion` comes up with 3 pods:

```bash
$ kubectl get statefulset zk-zk
NAME    DESIRED   CURRENT   AGE
zk-zk   3         3         1m20s
```

```bash
$ kubectl get pods
NAME                    READY   STATUS             RESTARTS   AGE
zk-zk-0                 1/1     Running            0          23s
zk-zk-1                 1/1     Running            0          23s
zk-zk-2                 1/1     Running            0          23s
```
