---
title: Getting Started
type: docs
weight: 1
---

# Getting Started

## Pre-requisites

Before you get started:

- Install Go `1.12.3` or later
- This project uses [Go Modules](https://github.com/golang/go/wiki/Modules). Set `GO111MODULE=on` in your environment.
- Kubernetes Cluster `1.13` or later (e.g. [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/))
- [Install kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) with version `1.13` or later
- [Install Kustomize](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/INSTALL.md) with version `2.0.3` or later

## Go Modules

> ⚠️ This project uses Go Modules. Due to the current state of code generation in [controller-tools](https://github.com/kubernetes-sigs/controller-tools) and [code-generator](https://github.com/kubernetes/code-generator), KUDO currently **must** be cloned into its `$GOPATH`-based location.

## Installation Instructions

- Get KUDO repo: `go get github.com/kudobuilder/kudo/`
- `cd $GOPATH/src/github.com/kudobuilder/kudo`
- `kubectl apply -f config/crds` to deploy the universal CRDs
- `make run` to run the Operator with local go environment or `make deploy | kubectl apply -f -` to install to your kubernetes cluster

### Notes on Minikube
If you plan on developing and testing KUDO locally via Minikube, you'll need to launch your cluster with a reasonable amount of memory allocated.  By default, this is only 2GB - we recommend at least 8GB, especially if you're working with applications such as [Kafka](/docs/examples/apache-kafka/).  You can start Minikube with some suitable resource adjustments as follows:

``` shell
minikube start --cpus=4 --memory=8192 --disk-size=40g
```

**Before** `kubectl apply -f config/crds` you will need to have:

 * minikube running

## Deploy your first Application

Follow the instructions in the [Apache Kafka example](/docs/examples/apache-kafka/) to deploy a Kafka cluster along with its dependency Zookeeper.
