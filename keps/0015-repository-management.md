---
kep-number: 0015
title: Repository Management
short-desc: Details on repositories, how to generate, update and manage repos
authors:
  - "@kensipe"
owners:
  - "@kensipe"
  - "@fabianbaier"
creation-date: 2019-07-24
last-updated: 2019-08-29
status: provisional
---

# Repository Management

## Table of Contents

* [Repository Management](#repository-management)
      * [Table of Contents](#table-of-contents)
      * [Summary](#summary)
      * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals)
      * [Proposal](#proposal)
         * [Create Operator Tarball](#create-operator-tarball)
         * [Create Repository Index](#create-repository-index)
         * [Create Local Repository](#create-local-repository)
         * [Update a Repository](#update-a-repository)
         * [CLI Repository Help](#cli-repository-help)
         * [User Stories](#user-stories)
         * [Risks and Mitigations](#risks-and-mitigations)
      * [Graduation Criteria](#graduation-criteria)

## Summary

There is a number of ways that an operator can be installed via KUDO.  The operator developer can install via the local file system (with a operator folder or tarball).  It is also possible to install via url to a tarball.  While [KEP-0010](0010-package-manager.md) defines the packaging structure, this kep defines:

* The structure of a repository
* Defining authoritative repositories: incubator & graduated
* Working with multiple repositories
* Packaging and Repository management tooling
* Configuration of repositories for the CLI
* CLI Tooling for search, packaging, and pushing to/from repository
* Define the rules and means to promote incubated operators to the graduated repository.

By providing a repository (or set of repositories), KUDO will provide a mechanism for administrators to easy discover and install KUDO operators.  [Helm has provided some prior art](https://helm.sh/docs/topics/chart_repository/) which we should align closely to from a mental model stand point.
It is useful to note that helm is [moving away from the custom repository model](https://blog.bacongobbler.com/post/2019-01-25-distributing-with-distribution/index.html) and towards OCI-compatible registries for a backend.  While this seems like a worthy ambition, it adds work that we should focus on for a phase 2 effort.   The first version of the repository should

## Motivation

We need a way for the operator author to update a repository with their operator.  We also need a way for the operator to promote an operator as being production ready. Finally we need a KUDO user to point to different repositories.  

### Goals

- Automate the creation of a operator tarball
- Automate the creation or updating of the repository index with a new operator tarball
- Define rules for incubator and graduated along with graduation tooling.
- Define how an admin can create their own repository
- Install operator from incubator instead of graduated repository

### Non-Goals

- Manage repos of non-KUDO applications (e.g. Helm Charts)
- External formats defined by [KEP0013](0013-external-specs.md)
- Repository cacheing

## Proposal

### Create Operator Tarball

An operator packaage as defined in [KEP0010](0010-package-manager.md).  The storage package at the repository is a tarball. We need a way to create this tarball in a uniformed way. KUDO shall a way to generate a tarball based on the standard file system layout.  As an example: `kubectl kudo package docs/examples/zookeeper/`.  This will create a `zookeeper-3.4.10.tgz` based on parsing the `operator.yaml` for version details. It would be best if this included linting to ensure that the static structure of the operator is correct. At this point, it will guarantee that operator folder has `operator.yaml` and `params.yaml` along with a templates directory.  It will create the following operator tarball `{operator.yaml:name}-{operator.yaml:version}.tgz`.

### Create Repository Index

KUDO needs the ability to create an index file for the repository. Something like `kubectl kudo repo index new-repo --url https://kudo-repo.storage.googleapis.com`. In this example, the `kudo repo index` is the command. The `new-repo` is a folder containing operator tarballs. The index file defined in [KEP0010](0010-package-manager.md) will be created using the url provided for links in the file.

Steps for creating an index file for a new operator looks like:
```
mkdir new-repo
kubectl kudo package docs/examples/zookeeper/ --destination=new-rep
kubectl kudo repo index new-repo --url https://kudo-repo.storage.googleapis.com
```

### Create Local Repository

For convenience it should be possible to have KUDO start a repository service with a provided path.  `kubectl kudo serve --repo-path ./repo` would create a local http service for the repo in `./repo`

### Update a Repository

Inspired by Helm, KUDO needs a way to take a repository index and merge it with another index for the purposes of adding operators to a repository.  Example: `kubectl kudo repo index --merge https://kudo-repo.storage.googleapis.com`. This will create a new index file locally based on operators present and merge the index file from the repo location provided.

### CLI Repository Help

`kubectl kudo  repo index --help`

### User Stories

- Allow pushing of new KUDO operator to the incubator repository
- Allow running of a CNAB bundle as a KUDO Operator
- Allow the running of an Operator as a KUDO Operator


### Risks and Mitigations


## Graduation Criteria

We need to define the rules for promoting incubator operators to the graduated repository.  At least one criteria would be some level of test coverage and passing in the Kubernetes environments defined in KEP-0004.
