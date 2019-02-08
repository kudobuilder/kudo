# Kudo

Kudo provides a declarative approach to building production-grade Kubernetes Operators covering the entire application lifecycle. 

## Getting Started

See the [Documentation](docs) with [Examples](config/samples).

![Quick Start](docs/gif/quickstart-0.1.0.gif)

## Pre-requisites

Before you get started:

- Install Go `1.11` or later
- Latest version of `dep`
- Kubernetes Cluster `1.12` or later (e.g. [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/))
- [Configure kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) 

## Installation Instructions

- Get kudo repo: `go get github.com/kudobuilder/kudo/`
- `cd $GOPATH/src/github.com/kudobuilder/kudo`
- `make install` to deploy universal CRDs
- `make run` to run the Operator with local go environment


## Concepts
- *Framework*: High-level description of a deployable application (e.g., Apache Kafka)
- *FrameworkVersion*: Specific version of a deployable application, including lifecycle hooks for deployments, upgrades, and rollbacks (e.g., Kafka version 2.4.1)
- *Instance*: Resource created to manage an instance of specific FrameworkVersion. Instances are pets and have the same name throughout its entire lifecycle. (e.g., Kafka 2.4.1 cluster with 3 brokers) 
- *PlanExecution*: Kudo-managed resource defining the inputs and status of an instance’s executable plans (e.g., upgrade kafka from version 2.4.1 -> 2.4.2)

## Deploy your first Application

Create a `Framework` object for Zookeeper
```bash
$ kubectl apply -f config/samples/zookeeper-framework.yaml
framework.kudo.k8s.io/zookeeper created
```

Create a `FrameworkVersion` for the Zookeeper  `Framework`

```bash
$ kubectl apply -f config/samples/zookeeper-frameworkversion.yaml
frameworkversion.kudo.k8s.io/zookeeper-1.0 created
```
 

Create an Instance of the Zookeeper
```
$ kubectl apply -f config/samples/zookeeper-instance.yaml
instance.kudo.k8s.io/zk created
```

When an instance is created, the default `deploy` plan is executed

```
$ kubectl get planexecutions
NAME                  AGE
zk-deploy-317743000   53s
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


## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](http://slack.k8s.io/)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-dev)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

[owners]: https://git.k8s.io/community/contributors/guide/owners.md
[Creative Commons 4.0]: https://git.k8s.io/website/LICENSE
