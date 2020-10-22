---
kep-number: 19
title: Versioning of Operator Packages
short-desc: Connection between the different versions that describe an operator
authors:
  - "@mpereira"
  - "@nfnt"
  - "@zmalik"
owners:
  - "@nfnt"
creation-date: 2019-11-05
last-updated: 2019-11-29
status: implementable
---

# Versioning of Operator Packages

## Table of Contents

<!-- TOC -->

- [Versioning of Operator Packages](#versioning-of-operator-packages)
  - [Table of Contents](#table-of-contents)
  - [Concepts](#concepts)
    - [KUDO version](#kudo-version)
    - [KUDO API version](#kudo-api-version)
    - [Operator](#operator)
    - [Operator user (user)](#operator-user-user)
    - [Operator developer (developer)](#operator-developer-developer)
    - [Package](#package)
    - [Package versions](#package-versions)
    - [Package Registry](#package-registry)
    - [Application](#application)
    - [Application version (app version)](#application-version-app-version)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
      - [Non-functional](#non-functional)
      - [Functional](#functional)
    - [Non-Goals](#non-goals)
  - [Current State](#current-state)
  - [Proposal](#proposal)
    - [Pros](#pros)
    - [Cons](#cons)
    - [Example hypothetical timeline for Apache Cassandra releases](#example-hypothetical-timeline-for-apache-cassandra-releases)
    - [Required and Suggested Changes](#required-and-suggested-changes)
    - [Alternative considered: Operator version independent from app version (largely, the current state of things)](#alternative-considered-operator-version-independent-from-app-version-largely-the-current-state-of-things)
      - [Pros](#pros-1)
      - [Cons](#cons-1)
    - [CLI UX](#cli-ux)
      - [Install latest version of the Kafka operator](#install-latest-version-of-the-kafka-operator)
      - [Install the latest Kafka operator with a specific Kafka version](#install-the-latest-kafka-operator-with-a-specific-kafka-version)
      - [Install a Kafka operator with a specific version](#install-a-kafka-operator-with-a-specific-version)
      - [Install a Kafka operator with a specific operator version](#install-a-kafka-operator-with-a-specific-operator-version)
      - [Upgrade Kafka operator](#upgrade-kafka-operator)
    - [Support multiple KUDO API versions in the package repository](#support-multiple-kudo-api-versions-in-the-package-repository)
    - [Naming of package tarballs](#naming-of-package-tarballs)
    - [Naming of `OperatorVersion` objects](#naming-of-operatorversion-objects)
    - [Semantic versioning](#semantic-versioning)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [User Stories](#user-stories)
    - [Operator developer](#operator-developer)
    - [Kubernetes Cluster Admin (KCA)](#kubernetes-cluster-admin-kca)
  - [Open Questions](#open-questions)
  - [Implementation History](#implementation-history)

<!-- /TOC -->

## Concepts

### KUDO version

A released version of KUDO (CLI, controller-manager), e.g. 0.7.0, 0.8.0, etc.

### KUDO API version

The internal version for the KUDO implementation, e.g. v1alpha1, v1beta1, v1, etc. These correspond to KUDO CRDs.

### Operator

A [Kubernetes Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) built on KUDO.

### Operator user (user)

Someone that installs or manages an _operator_.

### Operator developer (developer)

Someone that builds or maintains an _operator_.

### Package

An archive that contains all necessary files for running an _operator_.

### Package versions

Version identifiers that represents all of the underlying exact versions or minimum version requirements for an _operator_:

- KUDO API version
- Application version
- Operator revision
- Kubernetes version

### Package Registry

Storage accessible through the network (most commonly through the internet) containing multiple packages.

### Application

An underlying technology managed by an _operator_, e.g. Apache Cassandra, Apache Kafka, etc.

While it is possible that an _operator_ is composed of multiple _applications_, we assume that an _operator_ packages a "main" _application_ and additional _applications_ of a package are helpers for the main _application_.

### Application version (app version)

The version of an _application_, e.g. 3.11.5 (for Apache Cassandra), 2.3.0 (for Apache Kafka), etc.

Even though this version will more commonly follow [SemVer](https://semver.org/) for operators managing popular applications, it is ultimately outside of our control and therefore cannot be assumed to be [SemVer](https://semver.org/).

Even though the original app version may not be SemVer, we require operator developers to use a SemVer application version to ensure correct ordering.

## Summary

The different versions of components that constitute a _package_ are surfaced to users and developers so that they are able to build, maintain, query, manage and publish operators.

## Motivation

The dependencies between KUDO, _packages_ and _applications_ managed by _operators_ require the use of multiple versions to fully reify an operator.

All of the different versions specified in the ["package version" section](#package-version) need to be captured as part of the operator in a concise way that makes it easy for operator users to install specific operators for any of the available versions, possibly based on dependency version constraints (can I install the Kafka Operator version X on my Kubernetes cluster version Y with KUDO version Z?), and simple for operator developers to build and maintain operators.

### Goals

#### Non-functional

1. Provide an _easy_ experience for operator users to interact with packages and manage operators
1. Provide a _simple_ experience for operator developers to build and maintain operators and publish packages
1. Provide a solution that is general enough to allow for future backward and forward compatibility with regards to KUDO and operator development, while opinionated enough to minimize confusion
1. Provide a solution that maximizes the possibility for automation
1. Follow proven industry and academia best-practices

#### Functional

Allow operator developers to:

- Publish _packages_ to a _package registry_
- Release operator-related features and bug fixes to operators based on a non-latest _app version_

Allow operator users to:

- Install packages with a specific _package version_
- Install packages' _latest_ versions
- Query the available _package versions_ for a _package_
- Query the available _package versions_ for a _package_ that are compatible with a specific version of Kubernetes
- Query the available _package versions_ for a _package_ that are compatible with a specific version of the KUDO API
- Receive operator-related features and bug fixes on operators based on a non-latest _app version_

### Non-Goals

- Explain or define the scenario of multiple KUDO versions with possibly multiple KUDO API versions running on a single Kubernetes cluster
- Explain or define the planned implementation of operator<->operator dependency graphs (e.g., Kafka operator based on Kafka 2.3.1 requires a Zookeeper operator instance with version above 3.4.10)
- Explain or define the relationship between KUDO version and KUDO API version
- Explain or define implementations for transparent and automatic handling of multiple package registries from the KUDO CLI program

## Current State

An `operator.yaml` file provides versions for all components describing an operator:

- The _KUDO API version_ is specified in the `apiVersion` field
- The _KUDO version_ is specified in the `kudoVersion` field
- The _application version_ is specified in the `appVersion` field
- The _operator version_ is specified in the `version` field
- The minimum Kubernetes version is specified in the `kubernetesVersion` field

From these versions, currently only the `version` field can be used when installing packages with `kubectl kudo install`. Furthermore, a package repository index only contains `appVersion` and `version` fields in its entries. This limits the possible installation scenarios.

## Proposal

Allow users to filter packages by application version in addition to the operator version.
Package version resolution considers both application version as well as operator version. Operator versions are evaluated for each application version.

### Pros

- Puts the app version (what the user cares about) front and center
- Makes it trivial to have an "app version"-based backport/release strategy
- Makes it trivial to have an "app version"-based support strategy (e.g., an organization communicates official support for only the N-2 versions of an operator)

### Cons

- Visually more complex
- Makes it complex to know which operator-related features are available in which versions. Operator revision for different app versions are unrelated
- Upgrading from `3.11.4-0.1.2` to `3.12.0-0.1.2` might result in missing some operator-related features that were present in `3.11.4-0.1.2`, even if unlikely

### Example hypothetical timeline for Apache Cassandra releases

_Resolved version_ is the combination of _app version_ and _operator version_ used when ordering the package versions.

| Time | Resolved Version | App version | Operator version | KUDO API version | Comment                                    | Operator revision change |
| ---- | ---------------- | ----------- | ---------------- | ---------------- | ------------------------------------------ | ------------------------ |
| T0   | 3.11.4-0.1.0     | 3.11.4      | 0.1.0            | v1beta1          | Initial release based on Apache C\* 3.11.x | -                        |
| T1   | 3.11.4-0.2.0     | 3.11.4      | 0.2.0            | v1beta1          | Important bug fix in operator-related code | Minor bump               |
| T2   | 3.11.4-0.2.1     | 3.11.4      | 0.2.1            | v1beta1          | Small bug fix in operator-related code     | Patch bump               |
| T3   | 3.12.0-0.1.0     | 3.12.0      | 0.1.0            | v1beta1          | Apache C\* 3.12.x release                  | Reset (0.1.0)            |
| T4   | 3.11.4-0.3.0     | 3.11.4      | 0.3.0            | v1beta1          | Operator-related feature A added to 3.11.x | Minor bump               |
| T4   | 3.12.0-0.2.0     | 3.12.0      | 0.2.0            | v1beta1          | Operator-related feature A added to 3.12.x | Minor bump               |
| T5   | 4.0.0-0.1.0      | 4.0.0       | 0.1.0            | v1beta1          | Apache C\* 4.0.x release                   | Reset (0.1.0)            |
| T6   | 3.11.4-0.4.0     | 3.11.4      | 0.4.0            | v1beta1          | Operator-related feature B added to 3.11.x | Minor bump               |
| T6   | 3.12.0-0.3.0     | 3.12.0      | 0.3.0            | v1beta1          | Operator-related feature B added to 3.12.x | Minor bump               |
| T6   | 4.0.0-0.2.0      | 4.0.0       | 0.2.0            | v1beta1          | Operator-related feature B added to 4.0.x  | Minor bump               |
| T7   | 3.11.4-1.0.0     | 3.11.4      | 1.0.0            | v1               | KUDO API version change                    | Major bump               |
| T7   | 3.12.0-1.0.0     | 3.12.0      | 1.0.0            | v1               | KUDO API version change                    | Major bump               |
| T7   | 4.0.0-1.0.0      | 4.0.0       | 1.0.0            | v1               | KUDO API version change                    | Major bump               |

### Required and Suggested Changes

1. Require the (optional) _app version_ to be SemVer, as it takes precedence over an operator version if set.
2. Rename the (required) `version` field to `operatorVersion` to make it clear that it is for the _operator version_.

### Alternative considered: Operator version independent from app version (largely, the current state of things)

#### Pros

- Visually simple
- It can be known just from the package version whether or not it contains a certain operator-related feature or bug fix
- Version is a combination of the application and operator in a single SemVer.

#### Cons

- Operator users care about the application version, not a "synthetic" version
- Makes it complex to have an "app version"-based backport strategy. Customers might not be able to use operator based on newer app versions due to constraints outside of our control, but would still want to receive new operator-related features and bug fixes
- Continuing from the second "pro", it might be possible that subsequent package versions with different app versions don't contain an operator-related feature or bug fix that only makes sense for a specific app version (e.g., version N+1 based on app version M doesn't include an operator feature available on version N based on app version M+1 because the operator feature only makes sense for app version M)
- The progression of the package version is confusing with regards to the application version progression

  Example:

  | Time | Version | App version | Comment                                                     |
  | ---- | ------- | ----------- | ----------------------------------------------------------- |
  | T0   | 1.0.0   | 2.3.0       | Initial release based on app version 2.3.x                  |
  | T1   | 2.0.0   | 2.4.0       | Release based on app version 2.4.x                          |
  | T2   | 2.1.0   | 2.4.0       | New operator-related feature A for latest app version 2.4.x |
  | T2   | 1.1.0   | 2.3.0       | Back-port feature A for app version 2.3.x on demand         |
  | T3   | 1.1.1   | 2.3.1       | Bug fix release for app version 2.3.x                       |
  | T4   | 2.2.0   | 2.4.0       | Important bug fix B in operator code for latest 2.4.x       |
  | T4   | 1.2.0   | 2.3.1       | Back-port bug fix B in operator code for 2.3.x on demand    |
  | T5   | 2.2.1   | 2.4.0       | Small bug fix C in operator code for latest 2.4.x           |
  | T5   | 1.2.1   | 2.3.1       | Back-port bug fix C in operator code for 2.3.x on demand    |

In above example we know:

- What is the latest version of operator just by looking at `Version` column
- When a feature was introduced

The relationship between app and operator version is decided by the operator developer.
In the above case a major bump in version of app-version is a major bump in operator version also. But this should be up to the each operator.
If a base tech does big releases with minor versions, the operator developer can chose to bump the major version of operator with each minor release of base tech.

### CLI UX

For the CLI UX, the _app version_ is surfaced in addition to the _operator version_.

| Concept             | Flag               |
| ------------------- | ------------------ |
| Operator version    | --operator-version |
| Application version | --app-version      |

Assuming the following packages exist in the package registry:

| Operator | App Version | Operator Version |
| -------- | ----------- | ---------------- |
| kafka    | 2.3.0       | 1.0.0            |
| kafka    | 2.3.0       | 1.1.0            |
| kafka    | 2.3.0       | 1.1.1            |
| kafka    | 2.3.1       | 1.2.0            |
| kafka    | 3.0.0       | 1.0.0            |
| kafka    | 3.0.0       | 1.1.0            |
| kafka    | 3.0.1       | 1.1.0            |
| kafka    | 3.1.1       | 1.0.0            |

#### Install latest version of the Kafka operator

```
kudo install kafka
```

Installs `kafka` with app version `3.1.1` and operator versions `1.0.0`.

#### Install the latest Kafka operator with a specific Kafka version

```
kudo install kafka --app-version 2.3.0
```

Installs `kafka` with app version `2.3.0` and operator version `1.1.1`.

```
kudo install kafka --app-version 3.0.0
```

Installs `kafka` with app version `3.0.0` and operator version `1.1.0`.

#### Install a Kafka operator with a specific version

Let's say a user wants to use Apache Kafka `3.0.0` and even though there's a `3.0.0` with a `1.1.0` revision released, they want the operator with revision `1.0.0` due to reasons like:

- revision `1.1.0` introduced a bug for which there's still no released fixes
- revision `1.1.0` changed in a way that will require them to invest time and resources to adapt

```
kudo install kafka --app-version 3.0.0 --operator-version 1.0.0
```

#### Install a Kafka operator with a specific operator version

In the case that the `--operator-version` flag is used to select a specific _operator version_, there could be multiple _app versions_ for this _operator version_. In this case, KUDO will try to order the _app version_ and install the latest one. Because the _app version_ is required to be SemVer, it will always be possible to order these versions. In the case that _app version_ isn't set, this is treated like a 'v0.0.0', i.e. it will be treated like the oldest version available.

```
kudo install kafka --operator-version 1.1.0
```

Installs `kafka` with app version `3.0.1` and operator version `1.1.0`.

#### Upgrade Kafka operator

The `kudo upgrade` commands would look similar as the `kudo install` commands above.

### Support multiple KUDO API versions in the package repository

Through the `apiVersion` field in an `operator.yaml` it is already guaranteed that the operator will match a specific KUDO API version. However, this field isn't taken into account when installing packages from a repository. KUDO will try to install the latest version of the package even if the `apiVersion` doesn't match. This can be mitigated by adding the packages `apiVersion` to the repository index and filtering out all packages that don't have a matching version. This way, older versions of packages can still be part of the repository and keep working with older versions of KUDO.

For this, the following changes have to be made in KUDO:

- Add `apiVersion` to the package index by updating the `Metadata` struct in `pkg/kudoctl/util/repo` and filling these values in `pkg/kudoctl/util/repo.ToPackageVersion`. A `index.yaml` entry should then look like this:

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

- Update `pkg/kudoctl/util/repo.ParseIndexFile` to filter packages by API version. It should filter out packages whose `apiVersion` doesn't match the current `apiVersion` of KUDO.

### Naming of package tarballs

To avoid ambiguity the `appVersion` of a package will be part of the tarball name. E.g., the Kafka package described above will be named `kafka-2.3.0-0.2.0.tgz`. For this, changes in `kudoctl/packages/writer` are necessary. If `appVersion` isn't set, it won't be part of the tarball name.

### Naming of `OperatorVersion` objects

Currently, `OperatorVersion` objects are named "%operatorName%-%operatorVersion%". To avoid ambiguity, this will be updated to "%operatorName%-%appVersion%-%operatorVersion%", similar to the naming of package tarballs.

### Semantic versioning

`appVersion`, `kudoVersion` and `operatorVersion` follow the [Semantic Versioning Specification](https://semver.org/). The following filtering should be used when installing packages from a repository:

1. Filter by `apiVersion` of packages that match the one used by `kubectl kudo`
2. Filter by `kudoVersion` of packages that are smaller or equal than the one of `kubectl kudo`
3. Filter by `appVersion` of package if provided by user
4. Use latest or user-provided `operatorVersion` of package

### Risks and Mitigations

Because existing packages already set an `apiVersion` in their `operator.yaml` and the described change only affects repository indexes, this doesn't break any existing packages. Older versions of KUDO will still work with the new indexes because the additional `apiVersion` field isn't used when parsing the old `Metadata` struct.
Changing the `appVersion` to be required is a breaking change. From the packages in the community operators, most already set this field. Only the example operators "cowsay" and "first-operator" don't set this field. They would have to be updated. Given that these aren't usually used in production, this is an easy change.
Changing the naming of `OperatorVersion` is a breaking change. It will further limit the maximum allow length of operator names because the length of this string is limited.

## User Stories

### Operator developer

- As an operator developer, I want to publish an initial version of an operator, so that I can release new applications for Kubernetes cluster admins
- As an operator developer, I want to publish a new application version of an operator, so that I can update software to a new application version.
- As an operator developer, I want to publish a new operator version of an operator with an existing version, so that I can maintain LTS application versions while evolving the operator itself over time.
- As an operator developer, I want to pin specific application versions and operator versions to specific version of KUDO, so that I can ensure compatibility of operators with KUDO versions
- As an operator developer, I want to pin operators to Kubernetes API versions, so that I can ensure compatibility of operators with specific Kubernetes versions

### Kubernetes Cluster Admin (KCA)

- As a Kubernetes cluster admin (KCA), I want to get a listing of all application versions of operators of the latest operator version, so that I can see the availability of versions of applications
- As a KCA, I want to install an operator of the latest version, so that I can use the software in my cluster.
- As a KCA, I want to install an operator of a specific version, so that I can use LTS or otherwise non-current versions of software in my cluster
- As a KCA, I want to upgrade an operator to a new application version, so that I can take advantage of new versions of applications
- As a KCA, I want to upgrade an operator to a new operator version while maintaining the same application version, so that I can stay on LTS versions of applications while taking advantage of new operator-specific capabilities
- As a KCA, I want to know what application versions and operator versions are compatible with my versions of KUDO and Kubernetes, so that I can install the right version of the operator for my compatibility matrix.

## Open Questions

- It should be possible to specify a set of versions that a package can upgrade to or downgrade from. Where do we store this information?

- Regarding operator "bundles", it should be possible to have operators with multiple underlying applications. How should that be reflected in operator and package metadata (`operator.yaml` and `index.yaml` respectively)? And, even though this should be possible, should we encourage that "bundles" are split into their own operators where it makes sense? There should be future functionality for defining operator<->operator dependency graphs, which should be able to help with this?

## Implementation History

- 2019/11/05 - Initial draft. (@nfnt)
- 2019/11/18 - Changed scope to include all package versions. (@nfnt)
- 2019/11/21 - Added "concepts" section and expanded existing sections. (@mpereira)
- 2019/11/25 - Expanded the single semver section and its CLI UX. (@zmalik)
- 2019/11/29 - Updated after discussing the various versioning approaches. (@nfnt)
- 2020/01/06 - Updated, renaming `version` to `operatorVersion`. (@nfnt)
