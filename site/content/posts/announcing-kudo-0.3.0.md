---
title: Announcing KUDO 0.3.0
date: 2019-06-28
---

We are proud to announce the release of KUDO v0.3.0! This release focuses on core infrastructure inside of KUDO as the format for developing operators and running operators stabilizes.

## What is KUDO?

[Kubernetes Universal Declarative Operator (KUDO)](https://github.com/kudobuilder/kudo) provides a declarative approach to building production-grade Kubernetes Operators covering the entire application lifecycle. An operator is a way to package and manage a Kubernetes application using Kubernetes APIs. Building an Operator usually involves implementing a [custom resource and controller](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/), which typically requires thousands of lines of code and a deep understanding of Kubernetes. KUDO instead provides a universal controller that can be configured via a declarative spec (YAML files) to operate any workload.

## Release Highlights

### Switch from Framework to Operator
Framework was the term the project previously used to represent a service that KUDO "the operator" would be instrumented to be an operator for. The intention was to bring clarity by introducing a new term. It was thought that the new term would allow for more accuracy in technical communication by reduce the overloading of the term operator. For some it caused more confusion and almost always resulted in the need for explanation. We have abandon the term completely resulting in a **breaking** change to the CRDs.

All previous concepts and required objects using the word Framework have been replaced with Operator, resulting Operator and OperatorVersion in the type definition.

This will result in some overloading of the term operator, but aligns better with the eco-system. The running KUDO controller is an operator that manages operators built with KUDO's packaging format. In the future, this will be expanded with support for Helm charts (v0.4.0), other packaging formats, existing operators, and beyond. KUDO's mission is to enable easy, declarative day two operations and API management for complex applications, no matter how they were originally built.

### Package Format

[KEP-10](https://github.com/kudobuilder/kudo/blob/v0.3.0/keps/0010-package-manager.md) for the package manager is underway, with release providing 2 features; package repository and a package format.

The general format allows for a number of files, but starts with an `operator.yaml`.  A common layout might be:

```
operator.yaml
params.yaml
templates/
     <additional files>.yaml
```
The `operator.yaml` defines the operator and its plans.  `params.yaml` defines the params to be used through all templated files.  While the templated files are defined under the templates folder. The templated files are referenced in the `operator.yaml` file.  The zookeeper operator provides a [good example](https://github.com/kudobuilder/operators/tree/a1f4cf/repository/zookeeper/0.1.0).


These package files are group together in a folder or subfolder or can be tarred together in a `tgz` file providing a couple of ways to install an operator.

### Package Installation Options

Introduced with this release are 2 ways to install an operator to KUDO. The first is designed for operator developers which is to install from the local filesystem. This can be either a folder with the operator package files or a tarball of those files.

`kubectl kudo install ./kafka` or `kubectl kudo install kafka.tgz`

This release also introduces a repository which allows for automatic discovery of operators to be installed.
`kubectl kudo install kakfa`

**Note:**  The order of operation when installing an operator is to check the local filesystem first, then to query the repository.   

### Skip Instance

kudoctl install now supports `-skip-instance` flag which skips installing an Instance when installing Operator and OperatorVersions. This is useful in a testing scenario - we install all of the Operators and OperatorVersions but don't want to install the Instances until later.

### Test Harness
The test harness outlined in [KEP-8](https://github.com/kudobuilder/kudo/blob/v0.3.0/keps/0008-operator-testing.md) has been implemented.  The harness described in [testing.md](https://github.com/kudobuilder/kudo/blob/v0.3.0/docs/testing.md) provides a mechanism for operator developers to create end to end integration testing for their frameworks.  Introduced with the test harness is the `kind: TestAssert` to assert operator state.  [Example tests](https://github.com/kudobuilder/operators/tree/v0.3.0/repository/zookeeper/tests/zookeeper-upgrade-test) are provided for the zookeeper operator.

### Operator Dependency Management Removed (For Now)
The proper handling of operator dependencies is more complex than first imagined / implemented.  This fact was called out in [Issue 438](https://github.com/kudobuilder/kudo/issues/438).  For this reason we have removed the `all-dependencies` flag from `kudoctl`.  This feature needs to be designed through the KEP process and will be implemented properly in the future.

## Changelog

Additionally, the team closed dozens of issues related to bugs and performance issues.

To see the full changelog and the list of contributors who contribued to this release, visit [the Github Release](https://github.com/kudobuilder/kudo/releases/tag/v0.3.0) page.

## What's Next?

Now that KUDO v0.3.0 has shipped, the team will begin planning and executing on v0.4.0. The focus of v0.4.0 is to provide operator extensions to provide KUDO's sequencing logic to formats including Helm Charts and [CNAB](https://cnab.io) bundles. v0.4.0 will also focus on the operator release process for operators being released into the repository.

[Get started](/docs/getting-started) with KUDO today. Our [community](/docs/community) is ready for feedback to make KUDO even better!
