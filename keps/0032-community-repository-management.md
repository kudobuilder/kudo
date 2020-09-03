---
kep-number: 32
title: Community Repository Management
short-desc: Details on how to add operator to the community repository
authors:
  - "@nfnt"
owners:
  - "@nfnt"
creation-date: 2020-07-02
last-updated: 2020-08-31
status: implemented
see-also:
  - KEP-10
  - KEP-15
---

# Community Repository Management

## Table of Contents

- [Community Repository Management](#community-repository-management)
  - [Table of Contents](#table-of-contents)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Graduation Criteria](#graduation-criteria)
  - [Implementation History](#implementation-history)

## Summary

By default, every KUDO deployment installs operator packages from the community repository. As we encourage the KUDO community to add new operator packages to this repository, this KEP defines a workflow to add new operator packages to the community repository.

## Motivation

Currently, most operator packages in the community repository are provided from the `kudobuilder/operators` Git repository. This Git repository contains package definitions, documentation and sometimes tests for each operator. While it is convenient, to have this all in a single repository, this is challenging for larger or third-party operators:

- Large operators usually provide their own Docker images and/or additional dependencies. These have to be built, tested and deployed. The tools for this can't be provided by `kudobuilder/operators`.
- Some operator tests cannot be covered by the test tooling provided in `kudobuilder/operators` making it necessary to host the tests in a separate Git repository.
- Providing multiple versions of an operator is only possible by using separate folders. Teams developing operators might prefer different workflows (e.g. Git tags)

These requirements make it necessary for operator packages to be developed in a separate Git repository. As a result, the operator packages in `kudobuilder/operators` are copies of specific versions of the respective upstream Git repository. This approach has challenges as well, because bugs discovered in an operator need to be resolved in the upstream Git repository, not in `kudobuilder/operators`. I.e., there needs to be metadata to link to the upstream Git repository. By having some form of metadata describing upstream operator package sources, the same metadata can be used to describe other properties of a package, e.g. the maturity level.

### Goals

- Provide a simple workflow for upstream operator packages to get added to the community repository
- Remove the need to "host" copies of upstream operator packages in a Git repository
- Provide a mechanism to define upstream operator package sources

### Non-Goals

- Change the practice of creating a PR against a Git repository to add operator packages to the community repository
- Change operator package management as defined in [KEP-10](0010-package-manager.md) and [KEP-15](0015-repository-management.md)
- Integration with [ArtifactHub](https://artifacthub.io/)

## Proposal

Upstream operator developers still create PRs against a Git repository to add their operator packages to the community repository. This Git repository lists references to upstream operator packages instead of a full copy of an operator package. A reference can point to a Git repository or a package tarball. A version of an operator package is described by a specific tag of a Git repository or a URL pointing to an operator tarball of that release.

For example, consider an operator package developed at `github.com/example/example-operator` that has tagged operator versions `1.0.0`, `1.1.0` and the operator package in the `operator` folder. To add or update this operator package, the developers would create a PR referencing their upstream Git repository and the specific version, e.g. by adding a file `example-operator.yaml` like

```yaml
apiVersion: index.kudo.dev/v1alpha1
kind: Operator
name: Example Operator
gitSources:
  - name: git-repo
    url: github.com/example/example-operator.git
versions:
  - appVersion: "1.0.0"
    operatorVersion: "1.0.0"
    git:
      source: git-repo
      tag: "1.0.0_1.0.0"
      directory: operator
  - appVersion: "1.1.0"
    operatorVersion: "1.0.0"
    git:
      source: git-repo
      tag: "1.1.0_1.0.0"
      directory: operator
```

Another example for an operator package that is provides as a tarball. This package doesn't set the optional application version and provides a URL for the package tarball.

```yaml
apiVersion: index.kudo.dev/v1alpha1
kind: Operator
name: Example Operator
version:
  - operatorVersion: "0.9.0"
    url: example.org/example-operator-0.9.0.tgz
```

While metadata like `name`, `appVersion`, and `operatorVersion` are also present in the referenced operator package, it is helpful for debugging purposes to duplicate this information here. This metadata will be available even if resolving the actual operator package fails.

Once this PR is merged, CI tooling detects the new YAML file, clones the referenced upstream Git repository, checks out the tag, and adds the operator package in the specified folder to the existing index. This workflow is similar to [krew-index](https://github.com/kubernetes-sigs/krew-index). Of course, CI tests that don't update the community repository can run before the PR is merged. These tests include checking the referenced operator package for validity. Additional conformance testing can be added as well.

We can add more metadata to the YAML reference file. E.g., support for different upstream sources like Mercurial.

### Risks and Mitigations

- The current `kudobuilder/operators` Git repository contains some operator packages that don't have an upstream Git repository. If we want to keep them part of the community repository, we need to ensure that they keep getting hosted
- A repository index and individual operator packages already provide metadata, e.g. operator maintainers. We should use this data (if possible) instead of adding similar metadata fields to package references

## Graduation Criteria

- Provide a tool to update the community repository from a list of operator references
- Update `kudobuilder/operators` to the new workflow
- Ensure that existing operators that aren't hosted in a separate Git repository are still part of the community repository

## Implementation History

- 2020/07/02 - Initial draft (@nfnt)
- 2020/07/15 - Updated API after tests with a prototype. Removed maturity levels (@nfnt)
- 2020/08/31 - Changed status to 'implemented' with `kudobuilder/operators-index` (@nfnt)
