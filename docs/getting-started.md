---
title: Getting Started
type: docs
weight: 1
---

# Getting Started

## Pre-requisites

Before you get started:

- Install [Go 1.11](https://golang.org/) or later
- Latest version of [dep](https://golang.github.io/dep/)
- Kubernetes Cluster `1.12` or later (e.g. [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/))
- [Configure kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) 

## Installation Instructions

- Get KUDO repo: `go get github.com/kudobuilder/kudo/`
- `cd $GOPATH/src/github.com/kudobuilder/kudo`
- `make install` to deploy universal CRDs
- `make run` to run the Operator with local go environment

## Deploy your first Application

Follow the instructions in the [Apache Kafka example](/docs/examples/apache-kafka/) to deploy a Kafka cluster along with its dependency Zookeeper.