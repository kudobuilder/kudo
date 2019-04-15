---
title: Getting Started
type: docs
weight: 1
---

# Getting Started

## Pre-requisites

Before you get started:

- Install Go `1.11` or later
- Latest version of `dep`
- Kubernetes Cluster `1.12` or later (e.g. [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/))
- [Install kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) with version `1.12` or later
- [Install Kustomize](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/INSTALL.md) with version `2.0.3` or later

## Installation Instructions

- Get KUDO repo: `go get github.com/kudobuilder/kudo/`
- `cd $GOPATH/src/github.com/kudobuilder/kudo`
- `make install` to deploy universal CRDs
- `make run` to run the Operator with local go environment

**Before** `make install` you will need to have:
  * minikube running
  * `~/.git-credentials` must exist with git credentials. If you are using two-factor auth you will need a create a [personal access token](https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line)

## Deploy your first Application

Follow the instructions in the [Apache Kafka example](/docs/examples/apache-kafka/) to deploy a Kafka cluster along with its dependency Zookeeper.
