---
title: Getting Started
type: docs
weight: 1
menu: "docs"
aliases: ["/docs/"]
---

# Getting Started

## Pre-requisites

Before you get started:

- Kubernetes Cluster `1.13` or later (e.g. [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/))
- [Install kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) with version `1.13` or later

## Install KUDO into your cluster

- `kubectl create -f https://raw.githubusercontent.com/kudobuilder/kudo/v0.3.3/docs/deployment/00-prereqs.yaml`
- `kubectl create -f https://raw.githubusercontent.com/kudobuilder/kudo/v0.3.3/docs/deployment/10-crds.yaml`
- `kubectl create -f https://raw.githubusercontent.com/kudobuilder/kudo/v0.3.3/docs/deployment/20-deployment.yaml`

If you want to use the KUDO kubectl plugin, you can now follow the [CLI plugin installation instructions](https://kudo.dev/docs/cli/).

### Notes on Minikube

If you plan on developing and testing KUDO locally via Minikube, you'll need to launch your cluster with a reasonable amount of memory allocated. By default, this is only 2GB - we recommend at least 8GB, especially if you're working with applications such as [Kafka](/docs/examples/apache-kafka/). You can start Minikube with some suitable resource adjustments as follows:

```shell
minikube start --cpus=4 --memory=8192 --disk-size=40g
```

## Deploy your first Operator

Follow the instructions in the [Apache Kafka example](/docs/examples/apache-kafka/) to deploy a Kafka cluster along with its dependency Zookeeper.
