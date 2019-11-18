---
kep-number: 19
title: Versioning of Operator Packages
authors:
  - "@nfnt"
owners:
  - "@nfnt"
creation-date: 2019-11-05
last-updated: 2019-11-05
status: provisional
---

# Versioning of Operator Packages

## Table of Contents

* [Summary](#summary)
* [Motivation](#motivation)
  * [Goals](#goals)
  * [Non-Goals](#non-goals)
* [Proposal](#proposal)
  * [Support multiple API versions in the package repository](#support-multiple-api-versions-in-the-package-repository)
  * [Install a specific application version](#install-a-specific-application-version)
  * [Naming of package tarballs](#naming-of-package-tarballs)
  * [Semantic versioning](#semantic-versioning)
  * [Risks and Mitigations](#risks-and-mitigations)
* [Implementation History](#implementation-history)

## Summary

The different versions of components that describe an operator package are surfaced to users, so that they are able to install exactly specify which version can be installed. Furthermore, compatibility with existing packages for older APIs of KUDO is provided.

## Motivation

The dependencies between KUDO, operators packages and the application that is managed by the operator enforce the use of multiple versions to fully describe these dependencies. First, there is the dependency on a specific API version of KUDO. Also, a specific version of an application is provided by the package. And last, the operator itself may have multiple revisions. All of these versions need to be captured as part of the operator in a concise way that makes it easy for users to install specific operators for any of these versions.

### Goals

* Allow users to install packages from a package repository that provides packages for multiple KUDO API versions
* Allow users to install a specific version of an application and/or a specific revision of an operator
* Reject packages that don't have a matching KUDO API version

### Non-Goals

* Handle operator dependencies on specific Kubernetes versions

## Proposal

A `operator.yaml` file already provides versions for all components describing an operator:
  * The dependency on KUDO is provided by the `apiVersion` and `kudoVersion` fields
  * The bundled application version is provided by the `appVersion` field
  * The version of the operator is provided by the `version` field
  * The version of Kubernetes is provided by the `kubernetesVersion` field

From these versions, currently only the `version` field can be used when installing packages with `kubectl kudo install`. Furthermore, a package repository index only contains `appVersion` and `version` fields in its entries. This limits the possible installation scenarios.
At least the following installation scenarios should be supported:
  1. A package repository could contain operators for older `apiVersion` and than the currently released KUDO. A user should be able to install these operators with older versions of KUDO.
  2. A user should be able to install a specific `appVersion` of an operator

### Support multiple API versions in the package repository

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

### Install a specific application version

As `appVersion` is already part of the repository index, a command line flag `--app-version` will be added to `kubectl kudo install` to select a specific application version. Because this version is application specific and might not follow [semantic versioning](https://semver.org/), the absence of this command line flag will result in KUDO installing the latest `version` of a package, or the version indicated with the `--version` flag.

### Naming of package tarballs

To avoid disambiguities, the `appVersion` of a package will be part of the tarball name. E.g., the Kafka package described above will be named `kafka-2.3.0-0.2.0.tgz`. For this, changes in `kudoctl/packages/writer` are necessary.

### Semantic versioning

`kudoVersion` and `version` follow the [Semantic Versioning Specification](https://semver.org/). `appVersion` is application dependent and might not follow this specification. It should be treated as a free-form string without ordering. The following filtering should be used when installing packages from a repository:
  1. Filter by `apiVersion` of packages that match the one used by `kubectl kudo`
  2. Filter by `kudoVersion` of packages that are smaller or equal than the one of `kubectl kudo`
  3. Filter by `appVersion` of package if provided by user
  4. Use latest or user-provided `version` of package

### Risks and Mitigations

Because existing packages already set an `apiVersion` in their `operator.yaml` and the described change only affects repository indexes, this doesn't break any existing packages. Older versions of KUDO will still work with the new indexes because the additional `apiVersion` field isn't used when parsing the old `Metadata` struct.

## Implementation History

- 2019/11/05 - Initial draft.
- 2019/11/18 - Changed scope to include all package versions.
