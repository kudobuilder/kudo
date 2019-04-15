# KUDO

<img src="https://kudo.dev/images/kudo_horizontal_color.png" srcset="https://kudo.dev/images/kudo_horizontal_color@2x.png 2x" width="256">

[![CircleCI](https://circleci.com/gh/kudobuilder/kudo.svg?style=svg)](https://circleci.com/gh/kudobuilder/kudo)

Kubernetes Universal Declarative Operator (KUDO) provides a declarative approach to building production-grade Kubernetes Operators covering the entire application lifecycle.

## Getting Started

See the [Documentation](docs) with [Examples](config/samples).

![Quick Start](docs/images/quickstart-0.1.0.gif)

## Pre-requisites

Before you get started:

- Install Go `1.11` or later
- Latest version of `dep`
- Kubernetes Cluster `1.12` or later (e.g. [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/))
- [Install kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) with version `1.12` or later
- [Install Kustomize](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/INSTALL.md)

## Installation Instructions

- Get KUDO repo: `go get github.com/kudobuilder/kudo/`
- `cd $GOPATH/src/github.com/kudobuilder/kudo`
- `make install` to deploy universal CRDs
- `make run` to run the Operator with local go environment

**Notes:**
1. If `go get ...` is not functioning, an alternative is to:
  * `cd $GOPATH`
  * `mkdir -p src/github.com/kudobuilder`
  * `cd src/github.com/kudobuilder`
  * `git clone git@github.com:kudobuilder/kudo.git`
2. **Before** `make install` you will need to have:
  * minikube running (some of the tests run against it)
  * `~/.git-credentials` must exist with git credentials. If you are using two-factor auth you will need a create a [personal access token](https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line)
  * `lint` in $PATH which is provided by having `$GOPATH\bin` in `$PATH` as in `export PATH=$GOPATH/bin:$PATH`.

## Running the Server
In order to run KUDO, [kubebuilder](https://book.kubebuilder.io/getting_started/installation_and_setup.html) must be installed and in your `PATH` and CRDs must be installed.   CRDs are installed with `make install` or `make manifests` and can be confirmed with `kubectl get crds` resulting in something looking like:

```
  $ kubectl get crds
  NAME                            CREATED AT
  frameworks.kudo.k8s.io          2019-04-12T19:50:18Z
  frameworkversions.kudo.k8s.io   2019-04-12T19:50:18Z
  instances.kudo.k8s.io           2019-04-12T19:50:18Z
  planexecutions.kudo.k8s.io      2019-04-12T19:50:18Z
```

To run server:  `make run`

## Running the CLI

* To create CLI: `make cli`
* ensure `$GOPATH/src/github.com/kudobuilder/kudo/bin` is in your `PATH`

To run CLI: `kubectl kudo`

## Useful Tools
  * [kustomize](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/INSTALL.md)
  * [goimports](https://godoc.org/golang.org/x/tools/cmd/goimports)
  * [golint](https://github.com/golang/lint)

## Concepts
- *Framework*: High-level description of a deployable application (e.g., Apache Kafka)
- *FrameworkVersion*: Specific version of a deployable application, including lifecycle hooks for deployments, upgrades, and rollbacks (e.g., Kafka version 2.4.1)
- *Instance*: Resource created to manage an instance of specific FrameworkVersion. Instances are pets and have the same name throughout its entire lifecycle. (e.g., Kafka 2.4.1 cluster with 3 brokers)
- *PlanExecution*: Kudo-managed resource defining the inputs and status of an instance’s executable plans (e.g., upgrade kafka from version 2.4.1 -> 2.4.2)

## Deploy your first Application

Create a `Framework` object for Zookeeper
```bash
$ kubectl apply -f https://raw.githubusercontent.com/kudobuilder/frameworks/master/repo/stable/zookeeper/versions/0/zookeeper-framework.yaml
framework.kudo.k8s.io/zookeeper created
```

Create a `FrameworkVersion` for the Zookeeper  `Framework`

```bash
$ kubectl apply -f https://raw.githubusercontent.com/kudobuilder/frameworks/master/repo/stable/zookeeper/versions/0/zookeeper-frameworkversion.yaml
frameworkversion.kudo.k8s.io/zookeeper-1.0 created
```


Create an Instance of the Zookeeper
```bash
$ kubectl apply -f https://raw.githubusercontent.com/kudobuilder/frameworks/master/repo/stable/zookeeper/versions/0/zookeeper-instance.yaml
instance.kudo.k8s.io/zk created
```

When an instance is created, the default `deploy` plan is executed.

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


## Community, Discussion, Contribution, and Support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

### KUDO Weekly Community Meetings

Weekly meetings occur every Thursday at [3pm UTC](https://www.google.com/search?q=3pm+UTC)

You can discuss the agenda or reach the maintainers of this project at:

- [Slack](https://kubernetes.slack.com/messages/kudo/)
- [Mailing List](https://groups.google.com/d/forum/kudobuilder)

Quick links:

- [Agenda document](https://docs.google.com/document/d/1UqgtCMUHSsOohZYF8K7zX8WcErttuMSx7NbvksIbZgg)
- [Zoom meeting link](https://mesosphere.zoom.us/j/443128842)

### Code of Conduct

Participation in the Kudo community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

- [OWNERS](https://github.com/kudobuilder/kudo/blob/master/OWNERS)
- [Creative Commons 4.0](https://git.k8s.io/website/LICENSE)
