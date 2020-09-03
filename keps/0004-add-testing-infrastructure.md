---
kep-number: 4
title: Add Testing Infrastructure
short-desc: What, where, when and how we will test
authors:
  - "@runyontr"
owners:
  - "@runyontr"
  - "@fabian"
editor: TBD
creation-date: 2019-02-18
last-updated: 2010-02-18
status: implementable
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
      * [Operator tests](#operator-tests)
   * [Kubernetes clusters](0004-add-testing-infrastructure.md#kubernetes-clusters)
   * [CICD](#cicd)
      * [Pull Requests](#pull-requests)
      * [Main Branch](#main-branch)
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

As the complexity and scope of KUDO grows, it becomes impossible to manually validate exisiting operators and capabilities still function as expected. As part of a robust CICD pipeline, a testing suite needs to be developed that can validate each commit, pull request, or even deployment of KUDO on a cluster.

These testing packages should also assure conformance for the `kudobuilder/operators` repository to particular versions of KUDO.

[KEP-0008](https://github.com/kudobuilder/kudo/blob/main/keps/0008-operator-testing.md) outlines the design of a testing harness for validating KUDO and operators. This document outlines testing procedures and policies - e.g., what, where, when and how we will test.

## Motivation

### Goals

- Ensure validation of API objects is functioning correctly
- Ensure controllers execute known process flows correctly
- Reduce review time for code changes by not requiring reviewers to validate functionality of test cases
- Reduce developer time for code changes by providing tools to validate functionality
- Provide developers clear tooling for addition additional tests to infrastructure to validate bug fixes and new features

### Non-Goals

- 100% Test coverage

## Proposal

This is a two phase proposal.

### Tests

KUDO will be tested by several different test suites that validate individual components, KUDO, and operators.

#### Unit tests

Unit tests test a single component and do not require an Internet connection or the Kubernetes API. These should be fast, run on every pull request, and test that individual KUDO components behave correctly. Unit tests are written as standard Go tests.

All packages should incorporate unit tests that validate their behavior. Some examples of tests that should be included are:

- pkg/apis - Conformance of objects, serialization, validation
- pkg/controller - Controllers implement the correct process flows for changes in Instance objects, and creation of PlanExecutions
- pkg/util - Util functions behave as expected

Go coverage reports will be used to ensure that changes do not reduce test coverage.

#### Integration tests

Integration tests test the KUDO controller in its entirety. These tests can be run against either a real Kubernetes cluster or a local control plane. Integration tests can be written using the test harness designed in KEP-0008.

The integration tests will consist of a set of representative Operators and OperatorVersions that utilize real world KUDO features. Integration tests should be added for new KUDO capabilities at the discretion of the capability author or reviewers.

Integration tests will be hidden behind a Go build tag and will only run when the `integration` tag is specified.

#### End-to-end tests

End to end tests test the KUDO controller on a real Kubernetes cluster (either kubernetes-in-docker or any other Kubernetes cluster). These should exercise KUDO's features and workflows including installing operators from the repository. End-to-end tests can also exercise the CLI to test CLI-based workflows.

End-to-end tests will be triggered on merge to main, for release pull requests, and manually for pull requests containing risky or major changes.

#### Operator tests

Operator tests test that an operator works correctly. These require a full Kubernetes cluster to run and will be run in CI for the Operators repository using the latest released version of KUDO. Instead of running every test on every pull request, we will only run the tests that test the operator changed in any given pull request. Operator tests will also be run against main and release builds of KUDO to verify that KUDO changes do not break operators. Operators are tested using the KUDO test harness from KEP-0008.

Operator tests will also be run in CI for the KUDO repository manually after review, but prior to merging using `/test` command supported by Prow.

Operator tests live in the `kudobuilder/operators` repository, with the file structure defined in KEP-0010.

### Kubernetes clusters

It is important that tests are run against many different configurations of Kubernetes to ensure that KUDO and operators are compatible with common Kubernetes configurations and distributions.

Operator tests will be run against several different Kubernetes clusters:

- A local cluster using [kind](https://github.com/kubernetes-sigs/kind) or [k3s](https://github.com/rancher/k3s).
- A [GKE cluster](https://cloud.google.com/kubernetes-engine/).
- An [EKS cluster](https://aws.amazon.com/eks/).
- An [AKS cluster](https://docs.microsoft.com/en-us/azure/aks/).

These clusters can be started either as a part of CI jobs or maintained long term to be used across many CI jobs.

### CICD

CICD is accomplished by a combination of CircleCI and Github Actions enabled for the KUDO repo.

#### Pull Requests

All Pull Requests into main need to have the following checks pass. These should be ordered in fastest to slowest to reduce the time spent when/if failures occur.

0. Check author has signed CLA
1. If the user is not a contributor, a contributor must write `/ok-to-test` on the pull request before it will be triggered.
1. `go fmt` does not change anything
1. `make lint` passes.
1. All unit tests pass (with `-race` flag)
1. Dockerfile builds (this requires all dependencies in the vendor folder)
1. All integration tests pass.

Perform the same set of tests after merging main into the branch.

#### Main Branch

##### Tests

The main branch will run all the same tests that pull requests do as well as the complete set of operator tests.

##### Pushes

Once merged into main a build process occurs to deploy the latest image at `kudobuilder/controller:latest`

##### Schedule

Running the build nightly/frequently will increase the chance of finding flaky tests.

##### Base Image Change

Since we don't have any tests that validate the image works (no integration tests) this seems uneccessary.

#### Tags/Release

- All unit, integration, and operator tests are run.
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
- Integration tests are run on PRs to KUDO.
- Operator tests are run on KUDO releases.
- Operator tests are run on PRs to Operators.

## Implementation History

- Proposal KEP - 2019/02/18

## Infrastructure Needed [optional]

This depends on the particular tooling used:

Prow

- Kubernetes Cluster
- Manhours for Running Prow
