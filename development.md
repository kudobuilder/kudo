# Development Guideline

This document is the canonical source of truth for tools, dependencies and toolchain used to develop KUDO.

Please submit an issue if you:
* Notice a requirement that this doc does not capture.
* Find a different doc that specifies requirements (the doc should instead link here).


## Prerequisites

* All interactions including code and comments need to follow the [code of conduct](code-of-conduct.md)
* All Contributions must follow the [contribution guideline](CONTRIBUTING.md)

The contributions guideline provides details on copyright and process.  This document focuses on technical toolchain and local development.

## Toolchain

The follow are a list of tools needed or useful in order to build, run, test and debug locally.

* make - used to build and test locally
* go - KUDO is written in Go, we use go 1.13+ style of development
* [goreleaser](https://goreleaser.com/) - is used for the release process
* docker
* [kind](https://kind.sigs.k8s.io/) and/or [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/)
* [jq](https://stedolan.github.io/jq/)
* [yq](https://github.com/mikefarah/yq)
* Unix commands:  awk, sed, cut, tar, sleep, curl, tee
* git
* kubectl
* [ngrok](https://ngrok.com/) - This is currently needed for full debugging. We are looking at alternatives. It does require signing up for this service.
* [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) - Binaries for `kube-apiserver` and `etcd` from kubebuilder are used for integration testing.

## Supported Kubernetes versions

Use of Kubernetes APIs is limited to APIs supported by the current version of Kubernetes' `kube-apiserver` and two minor versions older at the time of a KUDO release. This is similar to the allowed version skew of `kubelet` and `kube-apiserver` following [Kubernetes' version skew policy](https://kubernetes.io/docs/setup/release/version-skew-policy/). E.g., for a version of KUDO released while Kubernetes 1.18 is current, only APIs that are supported in Kubernetes 1.16, 1.17 and 1.18 can be used.
Functionality using APIs that don't conform to this rule should be placed behind a feature gate.

## Running Locally

These details will show using kind as a local kubernetes cluster.  These instructions should work with minikube as well.
You have 2 options when running locally.
1. you can run the KUDO controller and webhooks in the cluster or
2. you can run the KUDO controller and webhooks outside the cluster for debugging

When running locally we assume that you will not be using a [CertManager](https://cert-manager.io/docs/) thus initialization is with `--unsafe-self-signed-webhook-ca`.

### Running KUDO locally in Cluster

When running locally in the cluster, we use the KUDO CLI code to deploy the manager. You will either need to launch a released KUDO manager which means you have to pick a version.  Example:
`go run cmd/kubectl-kudo/main.go init --unsafe-self-signed-webhook-ca --version 0.15.0 -w`

In this example, we are choosing a released version of KUDO which is compatiable with the current cli version.  The `-w` is to wait until this is running.
This command will initialize the cluster with all the prerequisite configurations for serviceaccounts, namespaces, etc.

If you want to run a non-released version of KUDO, then build the latest code with docker.  `make docker-build`.  This process will end with something like:
```
Successfully built 2eef6479680e
Successfully tagged kudobuilder/controller:cbe6e68c270221a69b7e1f48f8f22fc39ade5c47
```
When using kind, you can pre-load controller image:
```
kind load docker-image kudobuilder/controller:cbe6e68c270221a69b7e1f48f8f22fc39ade5c47
```
Using this `kind load` approach, it is necessary to configurate the controller manifest to pull the image only if not present.
```
go run cmd/kubectl-kudo/main.go init --unsafe-self-signed-webhook-ca --kudo-image  kudobuilder/controller:cbe6e68c270221a69b7e1f48f8f22fc39ade5c47 --kudo-image-pull-policy=IfNotPresent -w
```

The other approach is to retag and push this image:
```
 docker tag  kudobuilder/controller:cbe6e68c270221a69b7e1f48f8f22fc39ade5c47 kensipe/controller:latest
 docker push kensipe/controller:latest
```

Now run locally with `go run cmd/kubectl-kudo/main.go init --unsafe-self-signed-webhook-ca --kudo-image kensipe/controller:latest -w`

**note:** The name kensipe/controller:latest was used as a full functional example.  You should choose a name for which you have rights to push to docker hub.  Using kind and the image load technique, you can avoid the push then pull approach completely.

### Debugging KUDO running locally outside the Cluster

The most significant challenge for running KUDO locally outside the cluster is KUDO v0.14.0+ requires a [webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/).  This webhook configuration requires kubernetes to make a callback to the webhook.  This callback is accomplished via an https call requiring valid certificates to work.  Our current solution to this is to use ngrok which was created for this purpose.

These instructions assume you haven't already initialized the cluster with previously running the manager locally.  If you did you will need to run `kubectl delete MutatingWebhookConfiguration kudo-manager-instance-admission-webhook-config` for this to work.

Steps to run the KUDO manager outside the local cluster:

1. `make generate-manifests`
1. (separate term) `ngrok http 443`
1. `make dev-ready`
1. `make run`

The first step will build a local cache of the required manifests.  
For this approach it is necessary to have an account with ngrok and to run at 443.
The `make dev-ready` deploys all the required prerequisite configurations to the cluster.  It will fail if ngrok is not running. If this fails or you restart ngrok simply run: `make update-webhook-config`.
The step for `make run` will run the code locally... but you could also skip that step and run from Goland, VSCode or your favorite editor/debugger.