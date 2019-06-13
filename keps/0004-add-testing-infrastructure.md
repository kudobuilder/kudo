---
kep-number: 4
title: Add Testing Infrastructure
authors:
  - "@runyontr"
owners:
  - @runyontr
  - "@fabian"
editor: TBD
creation-date: 2019-02-18
last-updated: 2010-02-18
status: implementable
see-also:
replaces:
superseded-by:
---

# add-testing-infrastructure

## Table of Contents

* [Summary](#summary)
* [Motivation](#motivation)
   * [Goals](#goals)
   * [Non-Goals](#non-goals)
* [Proposal](#proposal)
   * [Tests](#tests)
      * [Unit tests](#unit-tests)
      * [Integration tests](#integration-tests)
      * [Framework tests](#framework-tests)
   * [Kubernetes clusters](keps/0004-add-testing-infrastructure.md#kubernetes-clusters)
   * [CICD](#cicd)
      * [Pull Requests](#pull-requests)
      * [Master Branch](#master-branch)
         * [Tests](#tests-1)
         * [Pushes](#pushes)
         * [Schedule](#schedule)
         * [Base Image Change](#base-image-change)
      * [Tags/Release](#tagsrelease)
   * [User Stories](#user-stories)
      * [Story 1](#story-1)
      * [Story 2](#story-2)
   * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)
* [Infrastructure Needed [optional]](#infrastructure-needed-optional)

## Summary

As the complexity and scope of KUDO grows, it becomes impossible to manually validate exisiting frameworks and capabilities still function as expected. As part of a robust CICD pipeline, a testing suite needs to be developed that can validate each commit, pull request, or even deployment of KUDO on a cluster.

These testing packages should also assure conformance for the `kudobuilder/frameworks` repository to particular versions of KUDO.

[KEP-0008](https://github.com/kudobuilder/kudo/blob/master/keps/0008-framework-testing.md) outlines the design of a testing harness for validating KUDO and frameworks. This document outlines testing procedures and policies - e.g., what, where, when and how we will test.

## Motivation

### Goals

- Ensure validation of API objects is functioning correctly
- Ensure controllers execute known process flows correctly
- Validate Framework and FrameworkVersions in kudobuilder/frameworks adhere to the API spec defined by Kudo. Provide common misconfigurations and validate that the testing framework notifies the users/developer of failure
- Reduce review time for code changes by not requiring reviewers to validate functionality of test cases
- Reduce developer time for code changes by providing tools to validate functionality
- Provide developers clear tooling for addition additional tests to infrastructure to validate bug fixes and new features

### Non-Goals

- 100% Test coverage

## Proposal

This is a two phase proposal.

### Tests

KUDO will be tested by several different test suites that validate individual components, KUDO, and frameworks.

#### Unit tests

Unit tests test a single component and do not require an Internet connection or the Kubernetes API. These should be fast, run on every pull request, and test that individual KUDO components behave correctly. Unit tests are written as standard Go tests.

All packages should incorporate unit tests that validate their behavior. Some examples of tests that should be included are:

- pkg/apis - Conformance of objects, serialization, validation
- pkg/controller - Controllers implement the correct process flows for changes in Instance objects, and creation of PlanExecutions
- pkg/util - Util functions behave as expected

Go coverage reports will be used to ensure that changes do not reduce test coverage.

#### Integration tests

Integration tests test the KUDO controller in its entirety. These tests can be run against either a real Kubernetes cluster or a local control plane. Integration tests will be manually run after review, but prior to merging using the [Circle CI manual approval feature](https://circleci.com/docs/2.0/triggers/#manual-approval). Integration tests can be written using the test harness designed in KEP-0008.

The integration tests will consist of a set of representative Frameworks and FrameworkVersions that utilize real world KUDO features. Integration tests should be added for new KUDO capabilities at the discretion of the capability author or reviewers.

Integration tests will be hidden behind a Go build tag and will only run when the `integration` tag is specified.

#### Framework tests

Framework tests test that a framework works correctly. These require a full Kubernetes cluster to run and will be run in CI for the Frameworks repository using the latest released version of KUDO. Instead of running every test on every pull request, we will only run the tests that test the framework changed in any given pull request. Framework tests will also be run against master and release builds of KUDO to verify that KUDO changes do not break frameworks. Frameworks are tested using the KUDO test harness from KEP-0008.

Framework tests live in the `kudobuilder/frameworks` repository, with the file structure defined in KEP-0010.

### Kubernetes clusters

It is important that tests are run against many different configurations of Kubernetes to ensure that KUDO and frameworks are compatible with common Kubernetes configurations and distributions.

Framework tests will be run against several different Kubernetes clusters:

- A local cluster using [kind](https://github.com/kubernetes-sigs/kind) or [k3s](https://github.com/rancher/k3s).
- A [GKE cluster](https://cloud.google.com/kubernetes-engine/).
- An [EKS cluster](https://aws.amazon.com/eks/).
- An [AKS cluster](https://docs.microsoft.com/en-us/azure/aks/).

These clusters can be started either as a part of CI jobs or maintained long term to be used across many CI jobs.

### CICD

Use [CircleCI](https://circleci.com/docs/) and the [GitHub Plugin](https://github.com/marketplace/circleci/plan/MDIyOk1hcmtldHBsYWNlTGlzdGluZ1BsYW45MA==#pricing-and-setup).

For OpenSource projects we will receive 1,000 monthly build minutes. With the test suite below, that should suffice as a baseline.

#### Pull Requests

All Pull Requests into master need to have the following checks pass. These should be ordered in fastest to slowest to reduce the time spent when/if failures occur.

0. Check author has signed CLA
1. `go fmt` does not change anything
1. `make check-formatting passes.
1. All unit tests pass (with `-race` flag)
1. Dockerfile builds (this requires all dependencies in the vendor folder)
1. All integration tests pass.

Perform the same set of tests after merging master into the branch.

#### Master Branch

##### Tests

The master branch will run all the same tests that pull requests do as well as the complete set of framework tests.

##### Pushes

Once merged into master a build process occurs to deploy the latest image at `kudobuilder/controller:latest`

##### Schedule

Running the build nightly/frequently will increase the chance of finding flaky tests.

##### Base Image Change

Since we don't have any tests that validate the image works (no integration tests) this seems uneccessary.

#### Tags/Release

- All unit, integration, and framework tests are run.
- Build process occurs to deploy the image at `kudobuilder/controller:latest`.
- Create the YAML to deploy Kudo, and package up to store in GitHub Release

### User Stories

#### Story 1

As a developer, I want to ensure my changes don't break existing functionality, even if I don't understand all the capabilities of Kudo.

#### Story 2

As a repository owner, I don't want to have to validate the execution of common plans/functionality as part of the review process.

### Risks and Mitigations

- Complexity of a test and CICD system
- Overlooking edge cases and assuming they're still working when tests pass

## Graduation Criteria

How will we know that this has succeeded?

- When repository owners can feel confident that code changes are not breaking functionality.
- Tests pass for the API objects
- Leverage testing scaffolding provided by (and subsequently removed by us) Kubebuilder for controller logic.

## Implementation History

- Proposal KEP - 2019/02/18

## Infrastructure Needed [optional]

This depends on the particular tooling used:

Prow

- Kubernetes Cluster
- Manhours for Running Prow

CircleCI/TravisCI/GoogleCloudBuild

- Free baseline. Paid if we get more usage.
