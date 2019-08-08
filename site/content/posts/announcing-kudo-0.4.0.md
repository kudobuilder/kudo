---
title: Announcing KUDO 0.4.0
date: 2019-07-31
---

We are proud to announce the release of KUDO v0.4.0! This release enhances the test harness and includes changes to make KUDO a good citizen, most notably the `apiVersion` fields were updated from the `k8s.io` domain to the `kudo.dev` domain.

## Release Highlights

### API Change to kudo.dev (Breaking)

All `apiVersion` fields were updated from k8s.io domain to kudo.dev domain. If you already have KUDO running in your cluster you have to remove all installed CRDs and operators and re-recreate CRDs and re-install operators as a part of upgrading to 0.4.0.

### Improvements in Test harness

There are a number of enhancements in the test harness mostly around resolving flaky tests and increasing usability for non-KUDO use-cases:

* Built-in support for launching and testing kind clusters.
* `--start-control-plane` no longer implies `--start-kudo`.
* The test harness supports minimal updates in test steps via patching.
* Namespaces set on resources in test steps are now respected.
* It is now possible to delete all resources of a given type in a test step.
* Various fixes related to handling of CRDs in tests.
* It is now possible to specify many manifest directories to install prior to running tests.

Read the [test docs](https://github.com/kudobuilder/kudo/blob/master/test/README.md) for more details.

### Global KUBECONFIG

`--kubeconfig` and `$KUBECONFIG` environmental configuration was moved to be a KUDO root configuration and is honored for all KUDO commands.

### Install Operator from URL

KUDO now allows for the installation of an operator from an URL to a tgz bundle. Installation previous supported installation from the local file system which was great for a developer and installation from the repository which is great after an operator is released. Installation from an URL is seen as a way to get help from testers prior to release.

To install:  `kubectl kudo install http://kudo.dev/zk.tar.gz`

### Remove interactive install

`--auto-approve` option from `kudo install` was removed. If you want to install operator without installing an instance, you can still use `--skip-instance`.

### Prefix our labels and annotations

To be a good citizens in the kubernetes ecosystem we now prefix all labels and annotations that are automatically populated by kudo with our namespace. For example instead of `operator` label we now use `kudo.dev/operator` label. The same goes for all other labels and annotations.

### Remove possibility to install multiple packages

It is not possible anymore to pass multiple arguments to `kudo install` command. Run multiple `kudo install` commands if you need to install more packages.

## Changelog

Additionally, the team closed dozens of issues related to bugs and performance issues.

To see the full changelog and the list of contributors who contribued to this release, visit [the Github Release](https://github.com/kudobuilder/kudo/releases/tag/v0.4.0) page.

## What's Next?

Now that KUDO v0.4.0 has shipped, we will shortly after follow with v0.5.0 containing features like kudo upgrade or test harness enhanced with ability to execute kubectl commands. After that, in 0.6.0, we will focus on implementing extensions as described in [KEP-12](https://github.com/kudobuilder/kudo/blob/master/keps/0012-operator-extensions.md) as well as improvements for observability and debugging.
See the [KUDO Roadmap](https://github.com/orgs/kudobuilder/projects/2) for details.

[Get started](/docs/getting-started) with KUDO today. Our [community](/community) is ready for feedback to make KUDO even better!
