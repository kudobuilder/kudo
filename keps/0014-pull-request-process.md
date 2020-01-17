---
kep-number: 0014
title: Pull Request Process
short-desc: Description of GitHub issues, labels, pull requests and commands
authors:
  - "@jbarrick-mesosphere"
owners:
  - "@jbarrick-mesosphere"
  - "@orsenthil"
editor: TBD
creation-date: 2019-07-16
last-updated: 2019-10-02
status: rejected
see-also:
  - KEP-0004
  - KEP-0008
---

# Pull Request Process

## Table of Contents

* [Summary](#summary)
* [Goals](#goals)
* [Proposal](#proposal)
   * [Issues and pull requests](#issues-and-pull-requests)
   * [Pull requests](#pull-requests)
   * [Test environment](#test-environment)
* [User Stories](#user-stories)
   * [Issue reporter](#issue-reporter)
   * [Outside contributor](#outside-contributor)
   * [Trusted contributor](#trusted-contributor)
   * [Repository administrator](#repository-administrator)
* [Infrastructure Needed](#infrastructure-needed)

## Summary

In order to ensure that contributors to the KUDO project have a good experience, we have a streamlined pull request review and merging process using Prow.

After analyzing prow for few iterations, the team decided to reject prow, and decided to adopt Github Actions for Pull Request Process.
Github Actions provided the desired goal as a service, and it was easier to administer for the project team members. The status of this proposal was changed to rejected.

## Goals

* Automate and enforce labeling of pull requests and issues (e.g., size).
* Automate pull request and issue life cycles.
* Automatically request pull request review from the correct people.
* Allow contributors to restrict who is allowed to run tests.
* Not be limited by public CI services (queue times, concurrency, resources).
* Use CNCF-standard tooling for our processes.

## Proposal

A Prow cluster will be deployed in Kubernetes for running CI and PR/issue automation. This cluster lives at https://prow.kudo.dev/ and will take action on Github as the `ci-mergebot` user. KEP-0004 outlines the infrastructure required.

### Issues and pull requests

Prow will be used to manage issue and pull request labels and life cycle. When an issue or pull request is created or commented on, it will be scanned for Prow commands (lines that begin with `/`).

Supported commands:

* `/help`: labels the issue or PR as wanting help from external contributors.
* `/good-first-issue`: labels the issue or PR as a good issue for new contributors to tackle.
* `/priority`: sets the priority on the issue or PR.
* `/milestone`: set a milestone for the issue or PR.
* `/lifecycle`: set a specific lifecycle label on the issue or PR.
* `/kind`: label the issue or PR as a specific kind (e.g., bug, enhancement, etc).
* `/close`: close an issue or PR .
* `/repoen`: open an issue on PR that has been closed.

When issues and PRs are 30 days old, they will be labeled `lifecycle/rotten` and closed automatically at 60.

### Pull requests

For pull requests only, Prow will automatically request review from the proper users, control when tests are run (and run them), report test status, and automatically merge the pull request when tests are passed and it has the proper approvals.

Supported commands:

* `/ok-to-test`: if the PR creator is an external contributor, Prow will not run tests on the pull request until a contributor has specified `/ok-to-test`.
* `/test`: re-run all of the tests or a specific test.
* `/retest`: re-run failed tests.
* `/override`: ignore a test's status.
* `/approve`: approve a pull request.
* `/lgtm`: mark a pull request as ready for merge, once it has the required number of approvals and all tests are passed, Prow will merge it automatically.
* `/cc`: request review from a specific user.
* `/hold`: prevent a pull request from being merged.

Once two contributors have marked `/approve` (or `/lgtm`, which implies `/approve`) and one types `/lgtm` and all tests pass, a pull request will be automatically merged.

Contributors are defined in `OWNERS` files in the repository. The `OWNERS` file is a YAML file containing valid approvers and reviewers:

```
approvers:
- user1

reviewers:
- user2
```

Users listed in `approvers` are allowed to approve a pull request. Users listed in `reviewers` are eligible for being automatically requested as reviewers by Prow on new pull requests (Prow will select two users from the list at random).

Prow will look for `OWNERS` files covering the directories being changed, e.g., the root `OWNERS` file applies to all changes, but `./tests/OWNERS` only applies to changes that in `./tests/`.

### Test environment

Tests and builds run in Prow which runs jobs as pods in Kubernetes. The pods are created with privileged mode enabled, so are able to use docker-in-docker to build, push, and run Docker images, and will have necessary credentials mounted in to support creating Kubernetes clusters in GCP and AWS.

Job logs and results are viewable via the Prow web interface and are permanently stored in Google Cloud Storage. Any files written to `$ARTIFACTS` in a Prow job are also uploaded to Google Cloud Storage where they are viewable. Additionally, any Junit files written to `$ARTIFACTS` are parsed and available for analysis in the Prow interface.

## User Stories

### Issue reporter

* As an issue reporter, I want transparency into when and how my issue will be triaged and addressed.

### Outside contributor

* As an outside contributor, I want reviews to be requested from relevant people without having to know much about the project.
* As an outside contributor, I want my pull request to automatically be merged when it is ready.
* As an outside contributor, I want to be able to run tests on my pull request.
* As an outside contributor, I want to be able to easily troubleshoot my test failures.

### Trusted contributor

* As a trusted contributor, I want tests to automatically run on my pull requests when I make them.

### Repository administrator

* As a repository administrator, I want to be able to control which users tests run automatically for.

## Infrastructure Needed

Prow cluster (KEP-0004).
