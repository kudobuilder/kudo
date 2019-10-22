---
kep-number: 19
title: Versioning of KUDO API in Operator Packages
authors:
  - "@nfnt"
owners:
  - "@nfnt"
creation-date: 2019-11-05
last-updated: 2019-11-05
status: provisional
---

# Versioning of KUDO API in Operator Packages

## Table of Contents

* [Summary](#summary)
* [Motivation](#motivation)
  * [Goals](#goals)
  * [Non-Goals](#non-goals)
* [Proposal](#proposal)
  * [Risks and Mitigations](#risks-and-mitigations)
* [Implementation History](#implementation-history)

## Summary

By adding a KUDO API version to package indexes, we will avoid that changes to the operator schema will affect existing KUDO versions and package definitions.

## Motivation

Operator packages are developed by defining the operator using YAML files. The schema to use for these files is defined by KUDO. While the KUDO team tries to keep the schema definitions consistent, there can be situations where the schema has to be changed. We need to ensure that packages are in sync with these changes while at the same time allowing existing packages to work with existing versions of KUDO.

### Goals

* Allow users to install packages from a package repository that provides packages for multiple KUDO API versions
* Reject packages that don't have a matching KUDO API version

### Non-Goals

* Handle operator dependencies on specific Kubernetes versions

## Proposal

Through the `apiVersion` field in an `operator.yaml` it is already guaranteed that the operator will match a specific API version of KUDO. However, this field isn't taken into account when installing packages from a repository. KUDO will try to install the latest version of the package even if the `apiVersion` doesn't match. This can be mitigated by adding the packages `apiVersion` to the repository index and filtering out all packages that don't have a matching version. This way, older versions of packages can still be part of the repository and keep working with older versions of KUDO.

For this, the following changes have to be made in KUDO:

* Add `apiVersion` to the package index by updating the `Metadata` struct in `pkg/kudoctl/util/repo` and filling these values in `pkg/kudoctl/util/repo.ToPackageVersion`. A `index.yaml` entry should then look like this:

```
  kafka:
  - appVersion: 2.3.0
    digest: e80c7b783d327190d489159e89e0a005a6a8b00610bdb7e8b1bea73c49bf485a
    maintainers:
    - email: zmalikshxil@gmail.com
      name: Zain Malik
    name: kafka
    urls:
    - https://kudo-repository.storage.googleapis.com/kafka-0.2.0.tgz
    version: 0.2.0
    apiVersion: kudo.dev/v1beta1
```

* Update `pkg/kudoctl/util/repo.ParseIndexFile` to filter packages by API version. It should filter out packages whose `apiVersion` doesn't match the current `apiVersion` of KUDO.

### Risks and Mitigations

Because existing packages already set an `apiVersion` in their `operator.yaml` and this change only affects repository indexes, this doesn't break any existing packages. Older versions of KUDO will still work with the new indexes because the additional `apiVersion` field isn't used when parsing the old `Metadata` struct. However, a user might have to manually specify a package version to make sure that their API versions match.

## Implementation History

- 2019/11/05 - Initial draft.
