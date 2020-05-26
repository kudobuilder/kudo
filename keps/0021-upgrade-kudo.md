---
kep-number: 21
short-desc: Upgrade an existing KUDO installation
title: KUDO upgrade
authors:
  - "@aneumann"
owners:
  - "@aneumann"
creation-date: 2019-11-28
last-updated: 2020-05-26
status: draft
---

# Upgrading KUDO

## Table of Contents
<!--ts-->
<!--te-->

## Summary

Upgrade an existing KUDO installation is required for wide adoption to KUDO. It allows the developers to make quick 
release cycles and users to keep up with the latest version. 

## Motivation

We strive for quick and regular updates of KUDO. We need a process for upgrading all the moving parts of KUDO, and how 
the different parts interact.

### Goals

- Provide means to upgrade an existing KUDO installation, including:
  - Prerequisites
  - CRDs (Changes to existing CRD version and migration)
  - The KUDO manager
- Verify the state of an existing KUDO installation

### Non-Goals

- Stable CRD API with multiple versions
- Updates/Upgrades of operators and operator versions itself
- Versioning of KUDO itself
- Multi-Tenancy, i.e. running multiple KUDO installations on the same cluster

## Proposal

- Extend the `kudo init` command to add a `--upgrade` flag that upgrades an existing KUDO installation
  - Without the `--upgrade` flag, `kudo init` will detect an existing installation and abort
  - This prevents a user to accidentally upgrade or overwrite an existing installation 
  - The upgrade process should verify the existing validation, do as much pre-verification as possible and then upgrade
    required prerequisites, CRDs, the controller, etc. If existing data structures as CRDs change between versions, 
    the upgrade process should automatically migrate the data
- Extend `kudo init` with a `--verify` flag that checks the existing installation and prints errors and warnings if
  the installation has any issues that diverge from the expected state
- Implement an e2e test harness to test KUDO upgrades
  - This harness is required to ensure our upgrade process works between different and new versions
  - The default test should install a previous version and test and upgrade to the current version

### User Stories [optional]

Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of the system.
The goal here is to make this feel real for users without getting bogged down.

#### Story 1

#### Story 2

### Implementation Details/Notes/Constraints [optional]

#### Pre-Upgrade Verification

Before any modification of the cluster is made, the CLI should check if the upgrade would cause any issues. This
includes 

What are the caveats to the implementation?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they releate.

#### Upgrade Process

#### Upgrade to CRDs

The upgrade process must never delete and recreate the CRDs - this would delete all existing custom resources of that
type. 

### Risks and Mitigations

What are the risks of this proposal and how do we mitigate.
Think broadly.
For example, consider both security and how this will impact the larger kubernetes ecosystem.

## Graduation Criteria

How will we know that this has succeeded?
Gathering user feedback is crucial for building high quality experiences and owners have the important responsibility of setting milestones for stability and completeness.
Hopefully the content previously contained in [umbrella issues][] will be tracked in the `Graduation Criteria` section.

[umbrella issues]: https://github.com/kubernetes/kubernetes/issues/42752

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling owner acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first KUDO release where an initial version of the KEP was available
- the version of KUDO where the KEP graduated to general availability
- when the KEP was retired or superseded

## Drawbacks [optional]

Why should this KEP _not_ be implemented.

## Alternatives [optional]

Similar to the `Drawbacks` section the `Alternatives` section is used to highlight and record other possible approaches to delivering the value proposed by a KEP.

## Infrastructure Needed [optional]

Use this section if you need things from the project/owner.
Examples include a new subproject, repos requested, github details.
Listing these here allows an owner to get the process for these resources started right away.