---
title: Getting Started
type: docs
weight: 1
menu: "docs"
aliases: ["/docs/"]
---

# Getting Started

## Pre-requisites

Before you get started using KUDO, you need to have a running Kubernetes cluster setup. You can use Minikube for testing purposes.

- Setup a Kubernetes Cluster in version `1.13` or later (e.g. [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/))
- Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) in version `1.13` or later.

### Notes on Minikube

If you plan on developing and testing KUDO locally via Minikube, you'll need to launch your cluster with a reasonable amount of memory allocated. By default, Minikube runs with 2GB - we recommend at least 8GB, especially if you're working with applications such as [Kafka](/docs/examples/apache-kafka/). You can start Minikube with some suitable resource adjustments as follows:

```bash
minikube start --cpus=4 --memory=10240 --disk-size=40g
```

## Install KUDO into your cluster

Once you have a running cluster with `kubectl` installed, you can install KUDO like so:

```bash
kubectl create -f https://raw.githubusercontent.com/kudobuilder/kudo/v0.5.0/docs/deployment/00-prereqs.yaml
kubectl create -f https://raw.githubusercontent.com/kudobuilder/kudo/v0.5.0/docs/deployment/10-crds.yaml
kubectl create -f https://raw.githubusercontent.com/kudobuilder/kudo/v0.5.0/docs/deployment/20-deployment.yaml
```

You can optionally install the `kubectl kudo` plugin, which provides a convenient set of commands that make using KUDO even easier. To do so, please follow the [CLI plugin installation instructions](https://kudo.dev/docs/cli/).

## Deploy your first Operator

Follow the instructions in the [Apache Kafka example](/docs/examples/apache-kafka/) to deploy a Kafka cluster along with its dependency Zookeeper.

## Create your first operator

To see the powers of KUDO unleashed in full, you should try [creating your own operator](/docs/developing-operators). 
