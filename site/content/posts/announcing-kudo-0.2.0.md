---
title: Announcing KUDO 0.2.0
date: 2019-06-07
---

We are proud to announce the release of KUDO v0.2.0! This release focuses on core infrastructure inside of KUDO as the format for developing operators and running operators stabilizes.

## What is KUDO?

[Kubernetes Universal Declarative Operator (KUDO)](https://github.com/kudobuilder/kudo) provides a declarative approach to building production-grade Kubernetes Operators covering the entire application lifecycle. An operator is a way to package and manage a Kubernetes application using Kubernetes APIs. Building an Operator usually involves implementing a [custom resource and controller](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/), which typically requires thousands of lines of code and a deep understanding of Kubernetes. KUDO instead provides a universal controller that can be configured via a declarative spec (YAML files) to operate any workload.

## Release Highlights

### Go Templating and Sprig

KUDO has switched to Go Templating with Sprig for templates. All Mustache templates should be replaced to their corresponding Go templates. The following keywords are available:

- `{{ .Name }}` - Name of the instance
- `{{ .Namespace }}` - Namespace the instance is located in
- `{{ .OperatorName }}` - Name of the operator
- `{{ .PlanName }}` - Name of the plan being run
- `{{ .PhaseName }}` - Name of the phase being run
- `{{ .StepName }}` - Name of the step being run
- `{{ .StepNumber }}` - Number of the step being run

Additionally, all parameters are now nested under `{{ .Params }}` in templates. For example, the Username parameter would be available as `{{ .Params.Username }}`.

KUDO has made the [Sprig](https://github.com/Masterminds/sprig) function library available in templates. This gives a wide range of functions available to operator developers. For safety, KUDO disables functions related to the environment and filesystem inside the manager container. In this release, this includes: `env`, `expandenv`, `base`, `dir`, `clean`, `ext`, `isAbs`.

Continued work will be done on this to move toward the format described in [KEP-9: Operator Toolkit](https://github.com/kudobuilder/kudo/blob/master/keps/0009-operator-toolkit.md).

### KUDO Registry on Google Cloud Storage

KUDO now uses GCS for the registry. This removes the need for `.git-credentials`, making the process for installing packages much simpler. Work will continue on this with [KEP-9: Operator Toolkit](https://github.com/kudobuilder/kudo/blob/master/keps/0009-operator-toolkit.md), [KEP-3: CLI](https://github.com/kudobuilder/kudo/blob/master/keps/0003-kep-cli.md), and the WIP KEP-10, which represents work going on in the KUDO Package Manager.

### Install Parameter Syntax changes

The KUDO Install CLI now uses a new syntax for setting parameters. To set a parameter, use:

```
kubectl kudo install kafka -p cpus=3
```

This is more consistent with tooling such as Helm.

### Homebrew Tap

A Homebrew tap is now available for the KUDO kubectl Plugin. To use it, simply tap the repo and install:

```
$ brew tap kudobuilder/tap
$ brew install kudo-cli
```

### Controller Distribution

The KUDO controller distribution is now documented in the documentation and contains the Kubernetes manifests needed to install KUDO into your cluster. The [Getting Started Guide](https://github.com/kudobuilder/kudo/blob/master/docs/getting-started.md) has more details. Work is ongoing on the KUDO site to incorporate this documentation and have "production ready" installation instructions for KUDO.

## Changelog

Additionally, the team closed dozens of issues related to bugs and performance issues.

To see the full changelog and the list of contributors who contribued to this release, visit [the Github Release](https://github.com/kudobuilder/kudo/releases/tag/v0.2.0) page.

## What's Next?

Now that KUDO v0.2.0 has shipped, the team will begin planning and executing on v0.3.0. The focus of v0.3.0 is to stabilize the packaging format for operator developers, as well as operator extensions to provide KUDO's sequencing logic to formats including Helm Charts and [CNAB](https://cnab.io) bundles.

[Get started](/docs/getting-started) with KUDO today. Our [community](/community) is ready for feedback to make KUDO even better!
