---
title: Getting Started
type: docs
weight: 1
---

# Getting Started for Developers

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
 * `~/.git-credentials` must exist with git credentials with details below.

### Setting up GitHub Credentials
In order to setup `~/.git-credentials` the file needs to have the format of:
```
https://<username>:<credential>@github.com
```

The username is your GitHub user name and the credential is your password. If you are using 2-factor authentication, the credentials will need to be an application [personal access token](https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line).


## Deploy your first Application

Follow the instructions in the [Apache Kafka example](/docs/examples/apache-kafka/) to deploy a Kafka cluster along with its dependency Zookeeper.

# Getting Started for Ops Team Members
If you are looking to get started but you do NOT have a development environment, this section is for you!  We assume you do NOT have go, make or other development tools.   Required tools include:

 - [Install kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) with version `1.13` or later
 - [Docker](https://docs.docker.com/v17.12/install/)

 **note:** This example also includes using [Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/))

 In order to get started from an ops perspective, it is necessary to do these steps:

 * Set the context of kubectl to work with the preferred cluster
 * Apply the KUDO CRDs to that cluster
 * Run the KUDO manager

## Applying CRDs
The CRDs can be installed using `kubectl apply -f config/crds` from the project.  If you are lacking git it is possible to pull a zip file of the project and apply for the unzipped folder.  An alternative is to apply from the url such as:

```
kubectl apply -f https://raw.githubusercontent.com/kudobuilder/kudo/master/config/crds/kudo_v1alpha1_framework.yaml
kubectl apply -f https://raw.githubusercontent.com/kudobuilder/kudo/master/config/crds/kudo_v1alpha1_frameworkversion.yaml
kubectl apply -f https://raw.githubusercontent.com/kudobuilder/kudo/master/config/crds/kudo_v1alpha1_instance.yaml
kubectl apply -f https://raw.githubusercontent.com/kudobuilder/kudo/master/config/crds/kudo_v1alpha1_planexecution.yaml
```

## Running Manager from Docker
The docker image beyond the ubuntu operating system only contains the manager native built binary.   It requires kubernetes the configurations necessary to connect with Kubernetes in order to run.   For convenience a `minikube-config` file is located under `/config` of the project.   The issue with the regular minikube configuration is that by default the file contains paths which contain usernames.   It is possible to use your own `~/.kube/config`, however you will need to mirror the path structures in the config file in mounted volumes.   The provided sample `minikube-config` simplifies this by expecting `.minikube` to be mounted in the `\root` path.

Assuming you have a local minikube running with the CRDs already applied.  Here is how you would run the dockerized manager against minikube.  This also assumes that you are running the command relative to `config/minikube-config`.

`docker run  -v $PWD/config/minikube-config:/root/.kube/config  -v $HOME/.minikube:/root/.minikube kudobuilder/controller:v0.1.0`
