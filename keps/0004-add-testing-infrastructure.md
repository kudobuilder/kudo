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

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [User Stories](#user-stories)
      * [Story 1](#story-1)
      * [Story 2](#story-2)
    * [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)
* [Infrastructure Needed [optional]](#infrastructure-needed-optional)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Summary

As the complexity and scope of KUDO grows, it becomes impossible to manually validate 
exisiting frameworks and capabilities still function as expected.  As part of a robust 
CICD pipeline, a testing suite needs to be developed that can validate each commit, pull 
request, or even deployment of KUDO on a cluster.

These testing packages should also assure conformance for the `kudobuilder/frameworks` repository 
to particular versions of KUDO.



## Motivation

### Goals

* Ensure validation of API objects is functioning correctly
* Ensure controllers execute known process flows correctly
* Validate Framework and FrameworkVersions in kudobuilder/frameworks adhere to the API spec defined by Kudo.  Provide common
  misconfigurations and validate the testing framework notifies the users/developer of failure
* Reduce review time for code changes by not requiring reviewers to validate functionality of test cases
* Reduce developer time for code changes by providing tools to validate functionality
* Provide developers clear tooling for addition additional tests to infrastructure to validate bug fixes and new features

### Non-Goals

* 100% Test coverage

## Proposal

This is a two phase proposal.  

### Tests

Add testing packages to:

* pkg/apis - Conformance of objects, serialization, validation
* pkg/controller - Controllers implement the correct process flows for changes in Instance objects, and creation of PlanExecutions
* pkg/util - Util functions behave as expected

Integration tests

* Identify and build a common set of Frameworks and FrameworkVersions that can demonstrate/validate that functionality of the controller
exists in real world situations.

### CICD

Use [CircleCI](https://circleci.com/docs/) and the [GitHub Plugin](https://github.com/marketplace/circleci/plan/MDIyOk1hcmtldHBsYWNlTGlzdGluZ1BsYW45MA==#pricing-and-setup).
For OpenSource projects we will recieve 1,000 monthly build minutes.  With the test suite below, that should suffice as a baseline.

#### Pull Requests

All Pull Requests into master need to have the following checks pass.  These should be ordered in fastest to slowest to reduce the time spent when/if failures occur

1) `go fmt` does not change anything 
2) `go lint` and `go vet` both pass
3) All unit tests pass (with `-race` flag)
4) Dockerfile builds (this requires all dependencies in the vendor folder)

Perform the same set of tests after merging master into the branch.

#### Master Branch

##### Pushes

Once merged into master a build process occurs to deploy the latest image at `kudobuilder/controller:latest`

##### Schedule

Running the build nightly/frequently will increaes the chancing of finding flakey tests.

##### Base Image Change

Since we don't have any tests that validate the image works (no integration tests) this seems uneccessary.


#### Tags/Release

* Build process occurs to deploy the image at `kudobuilder/controller:latest`.
* Create the YAML to deploy Kudo, and package up to store in GitHub Release


### User Stories

#### Story 1

As a developer, I want to ensure my changes don't break existing functionality, even if I don't understand all the capabilities
of Kudo.

#### Story 2

As a repository owner, I don't want to have to validate the execution of common plans/functionality as part of the review process.

### Implementation Details/Notes/Constraints [optional]

* Are new tests required for all (code) PRs?  If fixing a bug, it's re-assuring if you can provide a test that demonstrates it not working,
but the level of effort is significantly increased to push code.  This might be counter productive in trying to encourage
developers from contributing.
* Being able to run the "Test Frameworks" on a runninging cluster might provide value (e.g. sonobouy), but requires a maturation
of that capability and might not be worth the effort

### Risks and Mitigations

* Complexity of a test and CICD system
* Overlooking edge cases and assuming they're still working when tests pass

## Graduation Criteria

How will we know that this has succeeded?

* When repository owners can feel confident that code changes are not breaking functionality.
* Tests pass for the API objects
* Leverage testing scaffolding provided by (and subsequently removed by us) Kubebuilder for 
  controller logic.

## Implementation History

* Proposal KEP - 2019/02/18


## Infrastructure Needed [optional]

This depends on the particular tooling used:

Prow

* Kubernetes Cluster
* Manhours for Running Prow

CircleCI/TravisCI/GoogleCloudBuild

* Free baseline.  Paid if we get more usage.
