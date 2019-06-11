---
kep-number: 10
title: Package Manager
authors:
  - "@alenkacz"
  - "@fabianbaier"
  - "@gkleiman"
owners:
  - "@alenkacz"
  - "@fabianbaier"
  - "@gkleiman"
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
    * [Framework Developer](#framework-developer)
    * [Cluster Administrator](#cluster-administrator)
 * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
 * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)
* [Infrastructure Needed](#infrastructure-needed)

## Summary

As more and more developers seed our `kubebuilder/frameworks` repository with incredibly important Frameworks, there needs to be a provided structure that talks about how this repository is organized and made available. The underlying and agreed on structure also indirectly impacts or tangents to other KEPs, e.g. [KEP-0003 CLI](https://github.com/kudobuilder/kudo/blob/master/keps/0003-kep-cli.md), [KEP-0007 CLI Generation](https://github.com/kudobuilder/kudo/blob/master/keps/0007-cli-generation.md) or [KEP-0008 Framework Testing](https://github.com/kudobuilder/kudo/blob/master/keps/0008-framework-testing.md).

Overall, this KEP should capture how we plan to provide a great user experience of installing Frameworks with KUDO. Attention should be split to:

* How the Package structure looks like and dictates the overall repository structure
* What tools should be used to achieve this structure
* How Packages are distributed to the user

## Motivation

There are multiple Package managers and ideas out there and we need to decide which concept will work the best for KUDO in its current state. This is not an Install CLI subcategory KEP, but of course we need to think about the repercussions of decisions made in this KEP. There are multiple stakeholders directly being impacted by decisions made here, but this also paves the way to think about new problems we haven't articulated yet (e.g. how to provide KUDO Frameworks in air-gapped clusters, verification of Frameworks and their tarbundles, etc.).

### Goals

Specific goals are:
* Coming up with a structure for a Package
* Defining the overall `kubebuilder/framework` repository structure
* Having a solution on how to distribute this structure
* The solution should be platform-agnostic, so that Packages can be used on multiple platforms
* Adapting concepts from other successfull open-source projects in this regard

### Non-Goals

Non-goals are:
* Re-inventing the wheel
* Solving problems that are not related to the overall goals
* Some more that I can't think of yet

## Proposal

This is where we get down to the nitty gritty of what the proposal actually is. The folder structure and what could be interpreted as our v0 of the repository was originally discussed in PR [#87](https://github.com/kudobuilder/kudo/pull/87) and is a great way to start when thinking of this enhancement.

### User Stories

#### Framework Developer

As a Framework Developer I, ...

* would like to be able to host validation/testing plans and resources as part of the Package
* would like to update my Package version

#### Cluster Administrator

As a Cluster Administrator I, ...

* would like to be able to host my custom Framework extensions on a private KUDO Framework repo
* would like to be able to add additional validation/testing plans and resources

### Implementation Details/Notes/Constraints

The main interface to access Packages is a simple HTTP Server that can be easily accessed, even in air-gapped environments, depending on the user needs. As Storage Backends there could be a multitude of options such as `Local`, `Google`, `S3`, `Github`, `Minio`, `Docker` and so on.
Having a HTTP Server as an API that just serves essentially a single yaml file which holds all the information also helps to make hosting Packages very easy. For example, this works well with Google Cloud Storage as well as S3. All it requires, is the access to the yaml file served from a HTTP Server.

Some caveats to this could for instance be defining a clean interface that won't break when a user attempts to access Frameworks and hosted repos in later versions. As we are still building out this interface with our beta releases this must be expected, but as we get closer to our first stable versions we will have less and less breaking changes. Another caveat is deciding on the right proper structure in how a Package and its high level repo structure will look like. Design decisions here could potentialy impact future implementations of Storage Backends we haven't thought of yet. Another caveat would be identifying the right approach in versioning as this dictates a lot also how the structure will look like. Here we agreed on versioning for the `.tgz` files and Frameworks should be following the SemVer convention.

The Repo structure on the local laptop looks in the short term as follow:

```bash
/index.yaml
/kafka
/kafka/0.2.0
/kafka/0.2.0/kafka-framework.yaml
/kafka/0.2.0/kafka-frameworkversion.yaml
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

* This structure serves as the local repo, cache and source of truth

In the long term it will conform with KEP-0009 and have the following structure:

```bash
/index.yaml
/kafka
/kafka/2.2.0
/kafka/2.2.0/framework.yaml
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

For example, the `/kafka/2.2.0` folder (with whatever underlying structure) is zipped to `kafka-2.2.0.tgz`, where `2.2.0` is the current SemVer version of the package. 

The versioni of a Package ( e.g. `kafka-0.1.0` or `kafka-0.2.0` ) is not matching the current version of KUDO itself but follows its own SemVer timeline. The zipped Framework, then called Package, is made available through any HTTP Server.
 
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

We rely just on a HTTP Server, e.g. the out-of-the-box solution that Google Cloud Storage provides, that simply hosts the `tgz` files and provides them as downloads. The logic on keeping those single Frameworks in sync should live in the CLI and not this KEP itself. That also contributes to an abstraction layer that the HTTP server really doesn't need to be aware of all the business logic. 

The proposed structure is fairly easy to replicated and highly customizable.

Safety when distributing our Packages is another concern. As we continue working on this KEP we will add more details on how we can verify and prevent `Arbitrary software installation`, `Vulnerability to key compromises`, etc.. For now using HTTP-within-SSL/TLS is enough for verification.

### Risks and Mitigations

This potentially breaks our current CLI implementation on how to install a Framework and we probably need to also think about how we could mitigate it and/or make it backwards compatible. This also shows that we need to implement it in a way that changes are not backwards compatible. One solution to this is the HTTP server serving as a middle layer, which could also be a Single Point of Failure. For the example of our official hosted repo this is being minimized by relying on Google's infrastructure.

Other risks are in the way we distribute and install Packages without any validation/verification. As this KEP develops we will add more safety mechanisms such as sha checking.

## Graduation Criteria

Having a hosted implementation and being able to install Frameworks and getting metrics from it.
This includes e.g.:
* Solving the folder structure for Packages
* Solving the folder structure for the entire repository

## Implementation History

- 2019/05/16 - Initial draft.
- 2019/05/30 - Updates to structure from KEP-0009
- 2019/06/06 - Initial Re-Factoring to new Repo Structure ( https://github.com/kudobuilder/frameworks/pull/19 )

## Infrastructure Needed

This KEP requires us to have infrastructure which generates or updates our official Package`tgz` files and makes them available on our default Storage Backend. There is an interesting idea of having it entirely hosted e.g. on `Docker` or `Github` but this KEP should provide the proper conditions to be able to seemingly easy accomplish this.

Infrastructure that will be affected is:

* https://github.com/kudobuilder/frameworks
* Our CICD Pipeline for publishing Frameworks
* CLI needs to adopt the standards developed here