---
title: Usage
type: docs
menu: docs
---

# Workflow

This outlines an ideal workflow for both sys admins and application operators.

## SysAdmins

The definition for a particular version of a framework will be captured in the Spec of a Framework CRD object.

```bash
$  kubectl get frameworks
NAME          AGE
kafka         37m
hdfs          37m
zookeeper     37m
```

For each framework there will be a corresponding CustomResourceDefinition:

```bash
$ kubectl get crds -l type=framework
NAME                                AGE
kafkas.packages.kudo.k8s.io      37m
hdfs.packages.kudo.k8s.io        37m
zookeeper.packages.kudo.k8s.io   37m
```

And each framework will have different versions available (not actual kubectl call):

```bash
 $ kubectl get framework versions kafka
NAME            AGE
2.11-2.1.0      37m
2.10-2.1.0      37m
2.11-2.0.0      37m
2.10-2.0.0      37m
```

### Plans

The Plans for each application will be captured in the spec for each framework.

#### Deploy

The controller interprets the Spec provided for each framework as a list of Kubernetes objects. The default deploy plan creates all of the
Kubernetes objects and waits for them to become healthy. Multistep deploy plans break the set of kubernetes objects into groups and
waits to create the second group until the first group is created and healthy.

#### Upgrade

The controller interprets the Spec provided for each version of the framework into Kubernetes objects. For objects present in both versions,
the controller doesn't modify the objects if they're the same. The controller will create new objects in new versions and remove objects in previous
versions. Custom upgrade plans can adjust the order of deletion and creation.

To issue an upgrade, a command similar to this would be executed

```bash
kubectl patch kafka instance-name -p '{"spec": {"version":"2.11-2.1.0"}}'
```

### Parameters

Each instance of the framework will allow for customizations provided by parameters.

```bash
$ kubectl get framework parameters zookeeper
NAME                   Description
zookeeper.count        Number of Zookeeper nodes to be spun up
data.dir.size          Size of persistent volume used to store Zookeeper data
...
```

This gives an easy overview for Application Operators to understand how to configure the application.

## Application Operators

### Application Creation

To create an instance of a framework, a simple yaml file will be provided. This instance overrides the default
values by having 3 instances of Zookeeper in the `StatefulSet` and uses a larger than standard data directory.

```bash
cat <<EOF | kubectl create -f -
apiVersion: v1
kind: Zookeeper
metadata:
  name: zk
spec:
  parameters:
    zookeeper.count: 3
    data.dir.size: 10Gi
EOF

```

### Dependencies

When deploying an instance of a framework, values defined on a dependency will be applied automatically.

Create a Kafka instance that declares the above as a dependency:

```bash
apiVersion: packages.kudo.k8s.io/alpha
kind: Kafka
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: kafka
spec:
  # Add fields here
  name: "Kafka"
  dependency:
    zookeeper: zk
  parameters:
    broker.count: "1"
    zk.path: "/custom-path"
```

This will apply the `zookeeper.count` and `data.dir.size` parameters from the `zk` instance of Zookeeper.
