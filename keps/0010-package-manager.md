---
kep-number: 10
title: Package Manager
short-desc: KUDO Packages and basic repository description
authors:
  - "@alenkacz"
  - "@fabianbaier"
  - "@kensipe"
owners:
  - "@alenkacz"
  - "@fabianbaier"
  - "@kensipe"
editor: TBD
creation-date: 2019-05-16
status: implementable
---

# package-manager-for-distributing-kudo-packages

## Table of Contents

* [Table of Contents](#table-of-contents)
  * [Summary](#summary)
  * [Motivation](#motivation)
     * [Goals](#goals)
     * [Non-Goals](#non-goals)
  * [Proposal](#proposal)
     * [User Stories](#user-stories)
        * [Operator Developer](#operator-developer)
        * [Cluster Administrator](#cluster-administrator)
     * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
     * [Index File Specification](#index-file-specification)
     * [Risks and Mitigations](#risks-and-mitigations)
  * [Graduation Criteria](#graduation-criteria)
  * [Implementation History](#implementation-history)
  * [Infrastructure Needed](#infrastructure-needed)

## Summary

As more and more developers seed our `kubebuilder/operators` repository with incredibly important Operators, we need to define how these operators will be made available.

This KEP aims to define an operator repository structure. This structure indirectly impacts other KEPs, e.g., [KEP-0003 CLI](https://github.com/kudobuilder/kudo/blob/main/keps/0003-kep-cli.md), [KEP-0007 CLI Generation](https://github.com/kudobuilder/kudo/blob/main/keps/0007-cli-generation.md) or [KEP-0008 Operator Testing](https://github.com/kudobuilder/kudo/blob/main/keps/0008-operator-testing.md).

Overall, this KEP captures how we plan to provide a great user experience for installing Operators with KUDO. It focuses on:

* How the Package structure looks like and dictates the overall repository structure
* What tools should be used to achieve this structure
* How Packages are distributed to the user

## Motivation

There are multiple Package managers and ideas out there and we need to decide which concept will work the best for KUDO in its current state. This is not an Install CLI subcategory KEP, but of course we need to think about the repercussions of decisions made in this KEP. There are multiple stakeholders directly being impacted by decisions made here, but this also paves the way to think about new problems we haven't articulated yet (e.g. how to provide KUDO Operators in air-gapped clusters, verification of Operators and their tarbundles, etc.).

### Goals

* Coming up with a structure for a Package
* Defining the overall `kubebuilder/operator` repository structure
* Having a solution on how to expose a repository
* The solution should be platform-agnostic, so that Packages can be used on multiple platforms
* Adapting concepts from other successfull open-source projects in this regard

### Non-Goals

* Providing a solution for air-gapped clusters

## Proposal

The folder structure and what could be interpreted as our v0 of the repository was originally discussed in PR [#87](https://github.com/kudobuilder/kudo/pull/87) and is a great way to start when thinking of this enhancement.

### User Stories

#### Operator Developer

As an Operator Developer I, ...

* would like to be able to host validation/testing plans and resources as part of the Package
* would like to update my Package version

#### Cluster Administrator

As a Cluster Administrator I, ...

* would like to be able to host my custom Operator extensions on a private KUDO Operator repo
* would like to be able to add additional validation/testing plans and resources

### Implementation Details/Notes/Constraints

The main interface to access Packages is a simple HTTP Server that can be easily accessed, even in air-gapped environments, depending on the user needs. As Storage Backends there could be a multitude of options such as `Local`, `Google`, `S3`, `Github`, `Minio`, `Docker` and so on.

A repository will be served by a vanilla HTTP(S) server and consists of an index file in YAML and the `tgz` files for each operator. All this arquitecture requires is the ability to serve files using an HTTP server, which is very easy to implement using Google Cloud Storage or S3.

Some caveats of this architecture could for instance be defining a clean interface that won't break when a user attempts to use an older version of our tools to download an Operator hosted on a repo using a newer format. As we are still building out this interface with our beta releases this kind of breaking changes are to be expected, but as we get closer to our first stable versions we will have to minimize this kind of changes.

Deciding the right structure  of a Package and what the high level structure of the repository will look like. Design decisions here could potentially impact future implementations of Storage Backends we haven't thought of yet.

The operator versioning schema affects the structure of the repository. Here we agreed that versioning for the `.tgz` files and Operators should follow the SemVer convention.

A locally hosted repository will follow the following structure:

```bash
/index.yaml
/kafka
/kafka/0.2.0
/kafka/0.2.0/kafka-operator.yaml
/kafka/0.2.0/kafka-operatorversion.yaml
/kafka/0.2.0/kafka-instance.yaml
/kafka/0.2.0/metadata.yaml
/kafka/0.1.0
/kafka/docs
/kafka/docs/README.md
/kafka/docs/Demo.md
/kafka/tests
/kafka/tests/foobar.yaml
/kafka/tests/0.2.0/bar.yaml
/kafka/tests/0.1.0/foo.yaml
/zookeeper
...
```

* This structure serves as the local repo, cache and source of truth, and reflects the structure of the official repository hosted on GitHub.

In the long term it will conform with KEP-0009 and have the following structure:

```bash
/index.yaml
/kafka
/kafka/2.2.0
/kafka/2.2.0/operator.yaml
/kafka/2.2.0/params.yaml
/kafka/2.2.0/common
/kafka/2.2.0/common/common.yaml
/kafka/2.2.0/templates
/kafka/2.2.0/templates/deployment.yaml
/kafka/2.2.0/templates/...
/kafka/0.1.0
/kafka/docs
/kafka/docs/README.md
/kafka/docs/Demo.md
/kafka/tests
/kafka/tests/foobar.yaml
/kafka/tests/0.2.0/bar.yaml
/kafka/tests/0.1.0/foo.yaml
/zookeeper
...
```

The advantage of having a flat structure withing the hosted repo environment is, that for distribution the opinionated structure within the `.tgz` file is not much of importance and can be subject to change without breaking other assumptions.

For example, the `/kafka/2.2.0` folder (with whatever underlying structure) is zipped to `kafka-2.2.0.tgz`, where `2.2.0` is the current SemVer version of the Package.

The version of a Package (e.g., `kafka-0.1.0` or `kafka-0.2.0`) does not have to match the current version of KUDO itself but it follows its own SemVer timeline. The zipped Operator, called Package, is made available through any HTTP Server.

Our official repository is hosted on Google Cloud Storage and following a flat structure:

```bash
/index.yaml
/kafka-0.1.0.tgz
/kafka-0.2.0.tgz
/...
/kafka-1.0.0.tgz
/zookeeper-0.1.0.tgz
/...
/zookeeper-3.4.10.tgz
...
```

* This structure solely solves the distribution of Packages and is Storage-Backend agnostic, meaning it is possible to host this type of structure using other backends.

We rely on just an HTTP Server, e.g. the out-of-the-box solution that Google Cloud Storage provides, that serves operator `tgz` files and makes them available to users.

The logic for keeping the operators in sync should live in the CLI and is not defined on this KEP . That way the HTTP server only has to serve the index and the Package `tgz` files, without having to implement any business logic.

The proposed structure is fairly easy to replicate and highly customizable.

Safety when distributing our Packages is another concern. As we continue working on this KEP we will add more details on how we can verify and prevent `Arbitrary software installation`, `Vulnerability to key compromises`, etc. For now using HTTPS to fetch the index and Packages is considered sufficient.

### Index File Specification

`index.yaml` is the base definition of a repository.  It follows the following reference format.

```yaml
apiVersion: v1
entries:
  zookeeper:
  - name:  zookeeper
    version: "1.0.0"
    appVersion: "3.10.5"
    description: "description"
    maintainers: 
    - name: "Billy Bob"
      email: "bb@kudo.dev"
    digest: 94d436c2e7ee70c3b63c2b76b653f09fd326bc756a018e10f761261d17516eec
generated: "2020-02-21T14:02:36.57755-06:00"
```

An example looks like:
```yaml
apiVersion: v1
entries:
  elastic:
  - digest: 98beef6e771a64e42275b34059cde0bcf5244493a6511d1229bf3dd8f44c4791
    maintainers:
    - email: michael.beisiegel@gmail.com
      name: Michael Beisiegel
    name: elastic
    urls:
    - https://kudo-repository.storage.googleapis.com/0.7.0/elastic-0.1.0.tgz
    version: 0.1.0
  kafka:
  - appVersion: 2.3.0
    digest: e80c7b783d327190d489159e89e0a005a6a8b00610bdb7e8b1bea73c49bf485a
    maintainers:
    - email: zmalikshxil@gmail.com
      name: Zain Malik
    name: kafka
    urls:
    - https://kudo-repository.storage.googleapis.com/0.7.0/kafka-0.2.0.tgz
    version: 0.2.0
  - appVersion: 2.2.1
    digest: 3d0996ac19b9ff25c8d41f0b60ad686be8b1f73dd4d3d0139c6cdd1b1c4ae3e7
    maintainers:
    - email: zmalikshxil@gmail.com
      name: Zain Malik
    name: kafka
    urls:
    - https://kudo-repository.storage.googleapis.com/0.7.0/kafka-0.1.2.tgz
    version: 0.1.2
  - appVersion: 2.2.1
    digest: f576f92b0bd931a7792a0a0266865e8f20509c9b32b7f4d7d7b8856bf3bd1275
    maintainers:
    - email: zmalikshxil@gmail.com
      name: Zain Malik
    name: kafka
    urls:
    - https://kudo-repository.storage.googleapis.com/0.7.0/kafka-0.1.0.tgz
    version: 0.1.0
generated: "2019-09-16T10:26:23.331123-05:00"
```

### Risks and Mitigations

The changes proposed in this KEP are not necessarily backwards compatible and might prevent older versions of the CLI from installing Operators.

We could mitigate this risk by creating an HTTP service that translates from the new format to the old format. This HTTP service could be hosted on a reliable cloud provider such as Google Cloud.

Other risks are in the way we distribute and install Packages without any validation/verification. As this KEP develops we will add more safety mechanisms such as sha checking.

## Graduation Criteria

Having a hosted implementation and being able to install Operators and getting metrics from it.

This includes:
* Solving the folder structure for Packages
* Solving the folder structure for the entire repository

## Implementation History

- 2019/05/16 - Initial draft.
- 2019/05/30 - Updates to structure from KEP-0009
- 2019/06/06 - Initial Re-Factoring to new Repo Structure ( https://github.com/kudobuilder/operators/pull/19 )

## Infrastructure Needed

This KEP requires us to have infrastructure which generates or updates our official Package `tgz` files and makes them available on our default Storage Backend. There is an interesting idea of having it entirely hosted e.g. on `Docker` or `Github` but this KEP should provide the proper conditions to be able to seemingly easy accomplish this.

Infrastructure that will be affected is:

* https://github.com/kudobuilder/operators
* Our CICD Pipeline for publishing Operators
* CLI needs to adopt the standards developed here
