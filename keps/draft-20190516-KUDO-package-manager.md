---
kep-number: draft-05162019
title: KUDO Package Manager
authors:
  - "@fabianbaier"
owners:
  - TBD
  - "@fabianbaier"
editor: TBD
creation-date: 2019-05-16
status: provisional
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

As more and more developers seed our `kubebuilder/frameworks` repository with incredibly important Frameworks, there needs to be a provided structure that talks about how this repository is organized and made available. The underlying and agreed on structure also indirectly impacts or tangents to other KEPs, e.g. [KEP-0003 CLI](https://github.com/kudobuilder/kudo/blob/master/keps/0003-kep-cli.md), [KEP-0007 CLI Gernation](https://github.com/kudobuilder/kudo/blob/master/keps/0007-cli-generation.md) or [KEP-0008 Framework Testing](https://github.com/kudobuilder/kudo/blob/master/keps/0008-framework-testing.md).

Overall, this KEP should capture how we plan to provide a great user experience of installing Frameworks with KUDO. Attention should be split to:

* How the Package structure looks like and dictates the overall repository structure
* What tools should be used to achieve this structure
* How are packages are distributed to the user

## Motivation

There are multiple package managers and ideas out there and we need to decide which concept will work the best for KUDO in its current state. This is not an Install CLI subcategory KEP, but of course we need to think about the repercussions of decisions made in this KEP. There are multiple stakeholders directly being impacted by decisions made here, but this also paves the way to think about new problems we haven't articulated yet (e.g. how to provide KUDO Frameworks in air-gapped clusters, verification of Frameworks and their tarbundles, etc.).

### Goals

Specific goals are:
* Coming up with a structure for a Package
* Defining the overall `kubebuilder/framework` repository structure
* Having a solution on how to distribute this structure
* Having this platform-agnostic so Packages can be distributed across multiple solutions
* Adapting concepts from other successfull open-source projects in this regard

### Non-Goals

Non-goals are:
* Re-inventing the wheel
* Solving problems that are not related to the overall goals
* Some more that I can't think of yet

## Proposal

This is where we get down to the nitty gritty of what the proposal actually is. The folder structure and what could be interpreted as our v0 of the Registry was originally discussed in PR [#87](https://github.com/kudobuilder/kudo/pull/87) and is a great way to start when thinking of this enhancement.

### User Stories

#### Framework Developer

As a Framework Developer I, ...

* would like to be able to host validation/testing plans and resources as part of the package

#### Cluster Administrator

As a Cluster Administrator I, ...

* would like to be able to host my custom Framework extensions on a private KUDO Framework repo
* would like to be able to add additional validation/testing plans and resources

### Implementation Details/Notes/Constraints

We need to make sure that Packages can be easily accessed even if the underlying Storage Backend changed. Therefor, adding a middle layer like a registry server that has multiple Storage Backends as a swappable engine underneath seems like a good pattern. This also would enable users to keep track or restrict access to specific Frameworks.

The idea would be a HTTP Server that can be easily accessed, even in air-gapped environments, depending on the user needs. As Storage Backends there could be a multitude such as `Local`, `Google`, `S3`, `Github`, `Minio`, `Docker` and so on.
Having as an API a simple HTTP Server that just serves essentially a single yaml file that holds all the information also would enable us as a first iteration to use any HTTP server. This should work as it already with Google Cloud Storage or S3, all it requires is having access to the yaml file and something like the KUDO registry that serves like a middle layer inbetween could be easily added later on.

Some caveats to this could for instance be defining a clean interface that won't break things when a user attempts to access Frameworks and hosted repos. Another caveat is deciding on the right proper structure in how a Package and its higher repo structure will look like. Design decisions here could potentialy impact future implementations of Storage Backends we haven't thought of yet. Another caveat would be identifying the right approach in versioning as this dictates a lot also how the structure will look like.

The Repo structure on your local laptop could look in the short term as follow:

```bash
/index.yaml
/kafka
/kafka/2.2.0
/kafka/2.2.0/kafka-framework.yaml
/kafka/2.2.0/kafka-frameworkversion.yaml
/kafka/2.2.0/kafka-instance.yaml
/kafka/2.2.0/metadata.yaml
/zookeeper
...
```

* This structure serves as the local repo, cache and source of truth
* This also would be the on Github hosted official repo

In the long term it would look more like KEP-0009 conform e.g.:

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
/zookeeper
...
```

That would have the advantage that for distribution the opinionated structure within the `.tgz` file is not so much of importance and can be subject to change without breaking other assumptions.

The `/kafka/2.2.0` folder (with whatever underlying structure) would be zipped to `kafka-2.2.0.tgz` and made available through any HTTP Server (or later KUDO Registry with any Storage Backend), e.g. like on Google Cloud Storage:

```bash
/index.yaml
/kafka-0.1.0.tgz
/kafka-2.2.0.tgz
/zooekeper-3.4.10.tgz
...
```

* This structure would solely solve the distribution of Packages
* It should be Storage Backend agnostic, meaning possible to host this type of structure on most backends

We would have a specific HTTP Server, e.g. with the out-of-the-box solution that Google Cloud Storage provides already or later a KUDO Registry Server that knows on which Storage Backend the `tgz` files have been stored and provides them as downloads. Later, users should be able to download the entire repo structure as a `zip` as well (and not just single Frameworks). The logic on keeping those single Frameworks in sync should live in the CLI and not the KUDO Registry itself. That also contributes to an abstraction layer that the HTTP server really doesn't need to be aware of all this logic.

Again, this would be one way of implementing such scenario, however the proposed structure that lives behind a webserver seems to be fairly easy and highly customizable.

We should also think about the safety when distributing our Packages and how we can verify and prevent `Arbitrary software installation`, `Vulnerability to key compromises`, etc.. For now using HTTP-within-SSL/TLS is enough for verification.

### Risks and Mitigations

This potentially breaks our current CLI implementation on how to install a Framework and we probably need to also think about how we could mitigate it and/or make it backwards compatible. This also shows that we need to implement it in a way that changes are not backwards compatible. One solution to this is the HTTP server serving as a middle layer, which could also be a Single Point of Failure.

Other risks are in the way we distribute and install packages without any validation/verification.

We also need to make sure we are not having a Bottleneck by design and its impact to the larger ecosystem.

## Graduation Criteria

Having an hosted implementation and being able to install from it Frameworks and getting metrics from it.
This includes e.g.:
* Solving the folder structure for Packages
* Solving the folder structure for the entire registry


## Implementation History

2019/05/16 - Initial draft.
2019/05/30 - Updates to structure from KEP-0009


## Infrastructure Needed

This would also require us to have infrastructure that generates or updates `tgz` files and makes them available on our default Storage Backend. There is an interesting idea of having it entirely hosted e.g. on `Docker` or `Github` but this KEP should provide the proper conditions to be able to seemingly easy accomplish this.

Infrastructure that will be affected is:

* `kubebuilder/frameworks`
* Our CICD Pipeline for publishing Frameworks
* CLI needs to adapt the standards developed here