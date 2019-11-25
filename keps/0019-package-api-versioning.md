---
kep-number: 19
title: Versioning of Operator Packages
authors:
  - "@mpereira"
  - "@nfnt"
owners:
  - "@nfnt"
creation-date: 2019-11-05
last-updated: 2019-11-21
status: provisional
---

# Versioning of Operator Packages

**Table of Contents**

- [Concepts](#concepts)
    - [KUDO version](#kudo-version)
    - [KUDO API version](#kudo-api-version)
    - [Operator](#operator)
    - [Operator user (user)](#operator-user-user)
    - [Operator developer (developer)](#operator-developer-developer)
    - [Package](#package)
    - [Package version](#package-version)
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
- [Proposals](#proposals)
    - [With regards to package version and operator version](#with-regards-to-package-version-and-operator-version)
        - [Singular package/operator version (largely, the current state of things)](#singular-packageoperator-version-largely-the-current-state-of-things)
        - [Composite package/operator version](#composite-packageoperator-version)
    - [Support multiple KUDO API versions in the package repository](#support-multiple-kudo-api-versions-in-the-package-repository)
    - [Install a specific application version](#install-a-specific-application-version)
    - [Naming of package tarballs](#naming-of-package-tarballs)
    - [Semantic versioning](#semantic-versioning)
    - [Risks and Mitigations](#risks-and-mitigations)
- [Open Questions](#open-questions)
- [Implementation History](#implementation-history)

## Concepts

### KUDO version

A released version of KUDO (CLI, controller-manager), e.g. 0.7.0, 0.8.0, etc.

### KUDO API version

The internal version for the KUDO implementation, e.g. v1alpha1, v1beta1, v1, etc.

### Operator

An opinionated way to run a workload of a specific domain built on KUDO.

### Operator user (user)

Someone that installs or manages an _operator_.

### Operator developer (developer)

Someone that builds or maintains an _operator_.

### Package

An archive that contains all necessary files for running an _operator_.

### Package version

A single version identifier that represents all of the underlying exact versions or minimum version requirements for an _operator_:

- KUDO API version
- Application version
- Operator revision
- Kubernetes version

### Package Registry

Storage accessible through the network (most commonly through the internet) containg multiple packages.

### Application

An underlying technology managed by an _operator_, e.g. Apache Cassandra, Apache Kafka, etc.

It is possible that an _operator_ is composed of multiple _applications_.

### Application version (app version)

The version of an _application_, e.g. 3.11.5 (for Apache Cassandra), 2.3.0 (for Apache Kafka), etc.

Even though this version will more commonly follow [SemVer](https://semver.org/) for operators managing popular applications, it is ultimately outside of our control and therefore cannot be assumed to be [SemVer](https://semver.org/).

Programs dealing with the application version can try to parse it as a structured specification like SemVer, but should be able to handle anything (i.e., fall-back to strings).

## Summary

The different versions of components that constitute a _package_ are surfaced to users and developers so that they are able to build, maintain, query, manage and publish operators.

## Motivation

The dependencies between KUDO, _packages_ and _applications_ managed by _operators_ require the use of multiple versions to fully reify an operator.

All of the different versions specified in the ["package version" section](#package-version) need to be captured as part of the operator in a concise way that makes it easy for operator users to install specific operators for any of the available versions, possibly based on dependency version constraints (can I install the Kafka Operator version X on my Kubernetes cluster version Y with KUDO version Z?), and simple for operator developers to build and maintain operators.

### Goals

#### Non-functional

1. Provide an *easy* experience for operator users to interact with packages and manage operators
1. Provide a *simple* experience for operator developers to build and maintain operators and publish packages
1. Provide a solution that is general enough to allow for future backward and forward compatibility with regards to KUDO and operator development, while opinionated enough to minimize confusion
1. Provide a solution that maximizes the possibility for automation
1. Follow proven industry and academia best-practices

#### Functional

Allow operator developers to:

- Publish _packages_ to a _package registry_
- Release operator-related features and bug fixes to operators based on a non-latest _app version_

Allow operator users to:

- Install packages with a specific _package version_
- Install packages' *latest* versions
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

## Proposals

### With regards to package version and operator version

Two proposals: singular and composite package/operator version. In the case of potential operator "bundles" (operators containing multiple app versions) both would end up "hiding" those app versions behind the package/operator version.

#### Singular package/operator version (largely, the current state of things)

##### Pros

- Visually simple
- It can be known just from the package version whether or not it contains a certain operator-related feature or bug fix
- Version is a derived from the base tech and operator development as a single semver.

##### Cons

- Operator users care about the application version, not a "synthetic" version
- Makes it complex to have an "app version"-based backport strategy. Customers might not be able to use operator based on newer app versions due to constraints outside of our control, but would still want to receive new operator-related features and bug fixes
- Continuing from the second "pro", it might be possible that subsequent package versions with different app versions don't contain an operator-related feature or bug fix that only makes sense for a specific app version (e.g., version N+1 based on app version M doesn't include an operator feature available on version N based on app version M+1 because the operator feature only makes sense for app version M)
- The progression of the package version is confusing with regards to the application version progression

  Example:

  | Time | Version  | App version        | Comment                                                     |
  | ---- | -------- | ------------------ | ----------------------------------------------------------- |
  | T0   | 1.0.0    | 2.3.0              | Initial release based on app version 2.3.x                  |
  | T1   | 2.0.0    | 3.0.0              | Release based on app version 3.0.x                          |
  | T2   | 2.1.0    | 3.0.0              | New operator-related feature A for latest app version 3.0.x |
  | T2   | 1.1.0    | 2.3.0              | Back-port feature A for app version 2.3.x on demand         |
  | T3   | 1.1.1    | 2.3.1              | Bug fix release for app version 2.3.x                       |
  | T4   | 2.2.0    | 3.0.0              | Important bug fix B in operator code for latest 3.0.x       |
  | T4   | 1.2.0    | 2.3.1              | Back-port bug fix B in operator code for 2.3.x on demand    |
  | T5   | 2.2.1    | 3.0.0              | Small bug fix C in operator code for latest 3.0.x           |
  | T5   | 1.2.1    | 2.3.1              | Back-port bug fix C in operator code for 2.3.x on demand    |

In above example we know:
- What is the latest version of operator just by looking at `Version` column
- When a feature was introduced

The relationship between app and operator version is decided by the operator developer. 
In the above case a major bump in version of app-version is a major bump in operator version also. But this should be up to the each operator. 
If a base tech does big releases with minor versions, the operator developer can chose to bump the major version of operator with each minor release of base tech.   

#### Composite package/operator version

##### Pros

- Puts the app version (what the user cares about) front and center
- Makes it trivial to have an "app version"-based backport/release strategy
- Makes it trivial to have an "app version"-based support strategy (e.g., an organization communicates official support for only the N-2 versions of an operator)

##### Cons

- Visually more complex
- Makes it complex to know which operator-related features are available in which versions.  Operator revision for different app versions are unrelated
- Upgrades paths are unclear. Upgrading from `3.11.4-0.1.2` to `3.12.0-0.1.2` might result in missing some features that were present in `3.11.4-0.1.2`

##### Example hypothetical timeline for Apache Cassandra releases

Remember that _version_ is _operator version_ is _package version_.

| Time |      Version | App version | Operator revision | KUDO API version | Comment                                    | Operator revision change |
| ---- | ------------ | ----------- | ----------------- | ---------------- | ------------------------------------------ | ------------------------ |
| T0   | 3.11.4-0.1.0 |      3.11.4 |             0.1.0 | v1beta1          | Initial release based on Apache C\* 3.11.x | -                        |
| T1   | 3.11.4-0.2.0 |      3.11.4 |             0.2.0 | v1beta1          | Important bug fix in operator-related code | Minor bump               |
| T2   | 3.11.4-0.2.1 |      3.11.4 |             0.2.1 | v1beta1          | Small bug fix in operator-related code     | Patch bump               |
| T3   | 3.12.0-0.1.0 |      3.12.0 |             0.1.0 | v1beta1          | Apache C\* 3.12.x release                  | Reset (0.1.0)            |
| T4   | 3.11.4-0.3.0 |      3.11.4 |             0.3.0 | v1beta1          | Operator-related feature A added to 3.11.x | Minor bump               |
| T4   | 3.12.0-0.2.0 |      3.12.0 |             0.2.0 | v1beta1          | Operator-related feature A added to 3.12.x | Minor bump               |
| T5   |  4.0.0-0.1.0 |       4.0.0 |             0.1.0 | v1beta1          | Apache C\* 4.0.x release                   | Reset (0.1.0)            |
| T6   | 3.11.4-0.4.0 |      3.11.4 |             0.4.0 | v1beta1          | Operator-related feature B added to 3.11.x | Minor bump               |
| T6   | 3.12.0-0.3.0 |      3.12.0 |             0.3.0 | v1beta1          | Operator-related feature B added to 3.12.x | Minor bump               |
| T6   |  4.0.0-0.2.0 |       4.0.0 |             0.2.0 | v1beta1          | Operator-related feature B added to 4.0.x  | Minor bump               |
| T7   | 3.11.4-1.0.0 |      3.11.4 |             1.0.0 | v1               | KUDO API version change                    | Major bump               |
| T7   | 3.12.0-1.0.0 |      3.12.0 |             1.0.0 | v1               | KUDO API version change                    | Major bump               |
| T7   |  4.0.0-1.0.0 |       4.0.0 |             1.0.0 | v1               | KUDO API version change                    | Major bump               |

##### Required and Suggested Changes

1. Introduce the concept of the _operator revision_, which is an identifier that is re-set on every major/minor _app version_-based release. The "revision" naming is just to differentiate it from the monotonically increasing "version"
2. Make the concept of the _operator version_ be the composition of the _app version_ and an _operator revision_. In this section, _package version_ and _operator version_ are the same thing (and will henceforth be called _version_), like in the current state of things with the singular version, and are also deterministic.

   For example, in `2.3.0-1.0.0`:
   - The _operator version_ (and by extension the _package version_, or just _version_) is `2.3.0-1.0.0`
   - The _app version_ is `2.3.0`
   - The _operator revision_ is `1.0.0`
3. The _operator version_ wouldn't necessarily need to be explicitly set in operator or package metadata (`operator.yaml` and `index.yaml` respectively) given that the metadata contains fields for the _app version_ and the _operator revision_.

It's important to note that **operator revision for different app versions are unrelated**. E.g., assuming two versions `2.3.0-1.0.0` and `3.0.0-1.0.0`, even though both have the same operator revision (`1.0.0`) they wouldn't necessarily have any commonality with regards to exclusively operator-related features and bug fixes. The operator revision progression is only meaningful within an app version's `major.minor` family, i.e. `2.3.x` or `3.0.x`.

### CLI UX

Independent of the version strategy, this will be the CLI UX

| Concept                            | Flag          |
| ------------------------           | ---------     |
| Operator/package version (version) | --version     |
| Application version                | --app-version |
| Operator revision                  | --revision    |

Assuming the following packages exist in the package registry

for composite version:

| Operator | Version     |
| -------- | ----------- |
| kafka    | 2.3.0-1.0.0 |
| kafka    | 2.3.0-1.1.0 |
| kafka    | 2.3.0-1.1.1 |
| kafka    | 2.3.1-1.2.0 |
| kafka    | 3.0.0-1.0.0 |
| kafka    | 3.0.0-1.1.0 |
| kafka    | 3.0.1-1.1.0 |
| kafka    | 3.1.1-1.0.0 |

for single version:
                           
| Operator | Version  |
| -------- | -------- |
| kafka    | 1.0.0    |
| kafka    | 2.0.0    |
| kafka    | 2.1.0    |
| kafka    | 1.1.0    |
| kafka    | 1.1.1    |
| kafka    | 2.2.0    |
| kafka    | 1.2.0    |
| kafka    | 2.2.1    |
| kafka    | 1.2.1    |

**Install latest version of the Kafka operator**

In the composite package version strategy we will need to specify how app-version is ordered. 
```
kudo install kafka
```

**for composite version:** 

Installs `kafka-3.1.1-1.0.0`.

**for single version:** 

Installs `kafka-2.2.1`.

**Install the latest Kafka operator with a specific Kafka version**

**for composite version:**

```
kudo install kafka --app-version 2.3.0
```

Installs `kafka-2.3.0-1.1.1`.

```
kudo install kafka --app-version 3.0.0
```

Installs `kafka-3.0.0-1.1.0`.

**for single version:**

```
kudo install kafka --app-version 2.3.0
```

Installs `kafka-1.1.0`.

```
kudo install kafka --app-version 3.0.0
```

Installs `kafka-2.2.1`.

**Install a Kafka operator with a specific version**

Let's say a user wants to use Apache Kafka `3.0.0` and even though there's a `3.0.0` with a `1.1.0` revision released, they want the operator with revision `1.0.0` due to reasons like:
- revision `1.1.0` introduced a bug for which there's still no released fixes
- revision `1.1.0` changed in a way that will require them to invest time and resources to adapt

**for composite version:**

```
kudo install kafka --version 3.0.0-1.0.0
```

**for single version:**

it should be enough with single version
```
kudo install kafka --version 2.2.0
```

But users can also specify the `app-version` flag
```
kudo install kafka --version 2.2.0  --app-version 3.0.0
```

**Install a Kafka operator providing a partial version**

Another interesting thing that having the concept of _version_ contain the _app version_ concatenated with the _operator revision_ is that it would be possible to not have operator users necessarily have to know about the existence of _app version_ and _operator revision_. With regards to versioning, CLI interactions could optionally "flatten" the usage of both flags into just the `--version` flag, which could also be provided as a prefix to be matched.

**for composite version:**

```
kudo install kafka --version 3
```

Installs the latest Kafka operator based on Apache Kafka 3.x.x.

```
kudo install kafka --version 2.3
```

Installs the latest Kafka operator based on Apache Kafka 2.3.x.

```
kudo install kafka --version 2.3.1
```

Installs the latest Kafka operator based on Apache Kafka 2.3.1.

**for single version:**

```
kudo install kafka --app-version 3
```

Installs the latest Kafka operator based on Apache Kafka 3.x.x.

```
kudo install kafka --app-version 2.3
```

Installs the latest Kafka operator based on Apache Kafka 2.3.x.

```
kudo install kafka --app-version 2.3.1
```

Installs the latest Kafka operator based on Apache Kafka 2.3.1.

**Upgrade Kafka operator**

The `kudo upgrade` commands would look similar as the `kudo install` commands above.

**Search package registry**

**for composite version:**

```
$ kudo search kafka
kafka-2.3.0-1.0.0
kafka-2.3.0-1.1.0
kafka-2.3.0-1.1.1
kafka-2.3.1-1.2.0
kafka-3.0.0-1.0.0
kafka-3.0.0-1.1.0
kafka-3.0.1-1.1.0
kafka-3.1.1-1.0.0
```

```
$ kudo search kafka --version 3
kafka-3.0.0-1.0.0
kafka-3.0.0-1.1.0
kafka-3.0.1-1.1.0
kafka-3.1.1-1.0.0
```

```
$ kudo search kafka --version 3.1
kafka-3.1.1-1.0.0
```

```
$ kudo search kafka --version 2.3
kafka-2.3.0-1.0.0
kafka-2.3.0-1.1.0
kafka-2.3.0-1.1.1
kafka-2.3.1-1.2.0
```

**for single version:**

```
$ kudo search kafka
operator 	| app-version
1.0.0 		| 	2.3.0
1.1.0 		| 	2.3.0
1.1.1 		| 	2.3.1
1.2.0		| 	2.3.1
2.0.0 		| 	3.0.0
2.1.0 		| 	3.0.0
2.2.0 		| 	3.0.0
```

```
$ kudo search kafka --version 2
operator 	| app-version
2.0.0 		| 	3.0.0
2.1.0 		| 	3.0.0
2.2.0 		| 	3.0.0
```

```
$ kudo search kafka --app-version 2
operator 	| app-version
1.0.0 		| 	2.3.0
1.1.0 		| 	2.3.0
1.1.1 		| 	2.3.1
1.2.0		| 	2.3.1
```

```
$ kudo search kafka --app-version 2.3
operator 	| app-version
1.0.0 		| 	2.3.0
1.1.0 		| 	2.3.0
1.1.1 		| 	2.3.1
1.2.0		| 	2.3.1
```

---

An alternative notation for specifying the version could also look something like this, but this is not important:

```
kudo install kafka@3
```
```
kudo install kafka@2.3.0
```
```
kudo install kafka@2.3.1
```

And, orthogonal to the singular VS composite SemVer proposals:

### Support multiple KUDO API versions in the package repository

Through the `apiVersion` field in an `operator.yaml` it is already guaranteed that the operator will match a specific KUDO API version. However, this field isn't taken into account when installing packages from a repository. KUDO will try to install the latest version of the package even if the `apiVersion` doesn't match. This can be mitigated by adding the packages `apiVersion` to the repository index and filtering out all packages that don't have a matching version. This way, older versions of packages can still be part of the repository and keep working with older versions of KUDO.

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

Regardless of the operator/package version strategy chosen (singular VS composite), to avoid ambiguity the `appVersion` of a package will be part of the tarball name. E.g., the Kafka package described above will be named `kafka-2.3.0-0.2.0.tgz`. For this, changes in `kudoctl/packages/writer` are necessary.

### Semantic versioning

`kudoVersion` and `version` follow the [Semantic Versioning Specification](https://semver.org/). `appVersion` is application dependent and might not follow this specification. It should be treated as a free-form string without ordering. The following filtering should be used when installing packages from a repository:
  1. Filter by `apiVersion` of packages that match the one used by `kubectl kudo`
  2. Filter by `kudoVersion` of packages that are smaller or equal than the one of `kubectl kudo`
  3. Filter by `appVersion` of package if provided by user
  4. Use latest or user-provided `version` of package

### Risks and Mitigations

Because existing packages already set an `apiVersion` in their `operator.yaml` and the described change only affects repository indexes, this doesn't break any existing packages. Older versions of KUDO will still work with the new indexes because the additional `apiVersion` field isn't used when parsing the old `Metadata` struct.

## Open Questions

- It should be possible to specify a set of versions that a package can upgrade to or downgrade from. Where do we store this information?

- Regarding operator "bundles", it should be possible to have operators with multiple underlying applications. How should that be reflected in operator and package metadata (`operator.yaml` and `index.yaml` respectively)? And, even though this should be possible, should we encourage that "bundles" are split into their own operators where it makes sense? There should be future functionality for defining operator<->operator dependency graphs, which should be able to help with this?

## Implementation History

- 2019/11/05 - Initial draft. (@nfnt)
- 2019/11/18 - Changed scope to include all package versions. (@nfnt)
- 2019/11/21 - Added "concepts" section and expanded existing sections. (@mpereira)
- 2019/11/25 - Expanded the single semver section and its CLI UX. (@zmalik)
