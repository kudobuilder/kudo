---
title: Apache Zookeeper
type: docs
---

# Apache Zookeeper

Create a `Framework` object for Zookeeper
```bash
$ kubectl apply -f config/samples/zookeeper-framework.yaml
framework.kudo.k8s.io "zookeeper" created
```

Create a `FrameworkVersion` for the Zookeeper  `Framework`

```bash
$ kubectl apply -f config/samples/zookeeper-frameworkversion.yaml
frameworkversion.kudo.k8s.io "zookeeper-1.0" created
```
 

Create an Instance of Zookeeper
```
$ kubectl apply -f config/samples/zookeeper-instance.yaml
instance.kudo.k8s.io "zk" created
```

When an instance is created, the default `deploy` plan is executed

```
$ kubectl get planexecutions
NAME                  AGE
zk-deploy-392770000   11s
```

The statefulset defined in the `FrameworkVersion` comes up with 3 pods:

```bash
kubectl get statefulset zk-zk
NAME    DESIRED   CURRENT   AGE
zk-zk   3         3         1m20s
```

```bash
 kubectl get pods
NAME                    READY   STATUS             RESTARTS   AGE
zk-zk-0                 1/1     Running            0          23s
zk-zk-1                 1/1     Running            0          23s
zk-zk-2                 1/1     Running            0          23s
```