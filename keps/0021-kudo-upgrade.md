---
kep-number: 21
title: KUDO Upgrades
short-desc: Details how KUDO installations are upgraded
authors:
  - "@aneumann82"
owners:
  - "@aneumann82"
editor: TBD
creation-date: 2019-11-28
last-updated: 2020-09-22
status: implemented
---

# Upgrading KUDO

## Table of Contents
<!--ts-->
   * [Upgrading KUDO](#upgrading-kudo)
      * [Table of Contents](#table-of-contents)
      * [Summary](#summary)
      * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals)
         * [Current State](#current-state)
      * [Proposal](#proposal)
         * [Installed Operators](#installed-operators)
         * [KUDO Prerequisites](#kudo-prerequisites)
            * [Proposal for update process](#proposal-for-update-process)
         * [KUDO Manager](#kudo-manager)
            * [Proposal for update process](#proposal-for-update-process-1)
         * [CRDs](#crds)
            * [Proposal for update process](#proposal-for-update-process-2)
         * [KUDO CLI](#kudo-cli)
            * [Proposal for update process](#proposal-for-update-process-3)
      * [Updating KUDO installation](#updating-kudo-installation)
         * [Upgrade Steps](#upgrade-steps)
      * [User Stories](#user-stories)
            * [Story 1](#story-1)
            * [Story 2](#story-2)
         * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
         * [Risks and Mitigations](#risks-and-mitigations)
            * [Failure cases](#failure-cases)
      * [Infrastructure Needed](#infrastructure-needed)
      * [Resources](#resources)
      * [Implementation History](#implementation-history)

<!-- Added by: aneumann, at: Tue Sep 22 14:24:00 CEST 2020 -->

<!--te-->

## Summary

Implement upgrades for KUDO installation

## Motivation

We strive for quick and regular updates of KUDO. We need a process for upgrading all the moving parts of KUDO, and how 
the different parts interact, and what kind of compatibility we want to provide.

### Goals

- Specify how upgrades of KUDO are executed
  - Updates of prerequisites (Namespaces, RoleBindings, ServiceAccounts, etc.)
  - Updates of the CRDs
  - Updates of the KUDO manager
- Interoperability
  - If and how many multiple versions of CRDs are maintained
  - How a version of KUDO CLI work with older and newer CRD versions
- How to handle operators that are not supported by a new KUDO version

### Non-Goals

- Updates/Upgrades of operators and operator versions itself
- Supporting of multiple CRD versions
- Versioning of KUDO itself
- Multi-Tenancy, i.e. running multiple KUDO installations on the same cluster

### Current State

At the moment, KUDO does not provide any migration capabilities and needs a clean installation to use a new version.

## Proposal

- Extend the `kudo init` command to add a `--upgrade` flag that upgrades an existing KUDO installation
  - Without the `--upgrade` flag, `kudo init` should detect an existing installation and abort
  - The upgrade process should verify the existing validation, do as much pre-verification as possible and then upgrade required prerequisites, CRDs, the controller, etc.
- Implement an e2e test harness to test KUDO upgrades
  - This harness is required to ensure our upgrade process works between different and new versions

### Installed Operators

Before an upgrade can be performed, the CLI must validate that all installed operators are compatible with the new KUDO version.
To do this, each operator has certain attributes:
- `kudoVersion` This determines the minimum version of KUDO that is required for this operator
- `apiVersion` This defines the package format of the operator
  - Currently both `operator.yaml` and `params.yaml` have an `apiVersion`, and they should be kept in sync, even if they could be changed separately
  - The `apiVersion` is unrelated to the CRD version - the CLI converts from the package format to the CR, but the versions can change independently

As the `apiVersion` (and the package format) is only used to install new operatorVersions, it does not play into the compatibility concerns when upgrading KUDO. 

### KUDO Prerequisites
Expected update frequency: Medium  
Versioned: No, but closely tied to KUDO manager

The KUDO manager has a set of prerequisites that need to be present for the manager to run successfully. They are
the least likely to change, but probably the most specific. If we make changes here, we need to implement custom
migration code.

Usually the prerequisites are API resources, but may be more complex things.

- Prerequisites may be feature-gated
- Prerequisites may be k8s-version-dependent
- Prerequisites may have other prerequisites as dependencies (i.e., service account needs the namespace to exist)
- Prerequisites may have parameters and behave differently based on them (i.e. default namespace may be created, but a provided namespace needs to exist)

- Possible Prereqs
  - Namespace
  - Service Account
  - Role Bindings
  - Secrets
  - Webhooks
  - Other software in the cluster (i.e. cert-manager)

- For each prerequisite, there are a finite set of possible options in an update case:
  - The Prereq exists and has the same version/content as the new one - no action required
  - The Prereq does not exist - needs to be installed/created
  - The Prereq exists and has different version/content - needs to be updated/replaced
  - (The Prereq exists but should not exist anymore - needs to be removed/deleted) 

#### Proposal for update process 
Integrated into `kudo init --upgrade`

At the current time, it seems enough to use a simple install/update process.
  - The setup/update contains a list of all prerequisites in correct order
  - Each prereq validates the current installed state, and verifies that it can install/update the current state to the expected state
  - Prereqs that are deleted in newer versions need to stay in the list of prerequisites

We may arrive at a point where we need to implement custom migration logic for the prerequisites between KUDO versions. These can be implemented with the migration framework.

### KUDO Manager
Expected update frequency: High  
Versioned: Yes

The KUDO Manager is defined by an image version in a deployment set. To update, the deployment must be updated. The 
manager is closely tied to the CRDs, but not to the CLI. When CRDs are updated, the Manager will most likely also
need to be updated. 

#### Proposal for update process
Integrated into `kudo init --upgrade`

- Use semantic versioning for the manager binary
- As updates to the same CRD version should be backwards compatible, the manager could keep running while the CRD version is updated
  - To be on the very safe side, the manager should be in an `idle` state where no changes on operators are accepted or worked on
- After CRD update we can deploy the new manager version

### CRDs
Expected update frequency: Medium  
Versioned: Yes, with a CRD-Version

The CRDs are used to store installed operators, running instances and other custom persistent data. New features will regularly require us to add new
fields or even new CRDs.

#### Proposal for update process
Integrated into `kudo init --upgrade`

The upgrade process itself is simple:
- Update the CRDs. *Note:* This means an *update* to the CRDs, not a delete/recreate. If an existing CRD is deleted at any point in the update process, existing CustomResources of that type will be deleted as well.
- For now, only minor changes to the CRDs are possible with this approach.

In a future version KUDO will have to support multiple CRD versions. This will require installation of a conversion webhook 

- We need to provide support for multiple maintained API versions. This will allow us to evolve the API, and introduce backwards incompatible changes
  in a way that allows clients to migrate at their own pace.
- WebHook conversion will allow us to transparently switch to a new CRD version without manually migrating all existing CRs
  - WebHook conversion GA since 1.16 (1.13 alpha feature gate, 1.15 beta feature gate)
  - If we ever need more complex scenarios, i.e. splitting a CRD into two, or merging two CRDs into one, WebHook conversion will not cover this use case and we will need a different type of migration
- CRD versioning
  - Having an internal model of the data structures exposed via the API allows us to use defaulting, normalization between API versions and prevents  older clients from breaking existing resources
  - We can add new optional fields, and make other minor modifications to an existing API version.
  - More breaking changes (removing fields, making fields required, renaming fields, etc. ) require a CRD version change
  - See [K8s API Changes](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md) for details
- The introduction of new CRDs should be quite easy, as we simply need to install them
  - Updates of existing CRDs should be fine as well, adding new fields or full new versions of the CRD is quite simple
- Deletion of unused CRDs may be more difficult: We will need to make sure that all data is migrated from the old CRs
  - Deletion of a CRD deletes **all** CRs of that type, this is a high risk operation

### KUDO CLI
Expected update frequency: High

KUDO CLI is the command line tool to manage KUDO. It will be often updated to add new features and fix bugs. It needs
to be in sync with the installed CRDs, as it's writing them directly with the K8s extension API.

- Do we allow an older KUDO CLI to be used with a newer KUDO installation?
  - No. We would run into problems with the old KUDO CLI silently removing new fields from the CRDs
- Do we allow a newer KUDO CLI to be used with older KUDO installation?
  - Yes. If we want to prevent users having to install multiple KUDO versions, we need to support this. 
  - We need to decide how long and what version range of older KUDO installations we want to support.
  - Having to maintain multiple CRD versions inside KUDO CLI may be difficult. We would need to have checks on a new features if the feature is supported by an old KUDO installation.

#### Proposal for update process
User has to download newest KUDO version, either manually or via `brew` or other means.

- CLI must be at at least the version of the installed KUDO manager. It will not be possible to use an old KUDO CLI on a newer cluster
- CLI updates should be easy, therefore no need to introduce additional complexity here

## Updating KUDO installation

The update of a KUDO installation is triggered by  `kudo init --upgrade`

### Upgrade Steps
- Pre-Update-Verification
  - Detect if permissions to modify prerequisites are available
  - Verify that all prerequisite upgrade steps can be executed
  - Verify all old CRDs can be read by new KUDO version
  - Verify all installed operators are supported by new KUDO version
  - Dry-Run all available migrations to verify they can be executed
  - User can abort here
- Shutdown old manager version (or move it to an `idle` state)
- Update CRDs
- Run migrations
- Run prerequisite upgrades
- Install new manager
 
## User Stories

#### Story 1

An Operator wants to upgrade KUDO to the latest version to utilize new features. All installed operators should continue
to work.

#### Story 2

An Operator manages two K8s clusters with different KUDO versions installed. How does he manage to control both in the
most easy way?

### Implementation Details/Notes/Constraints

### Risks and Mitigations
This operation will **need** a `--dry-run` option
- Do normal Pre-Update-Verification
- Read all existing CRs and run migration to new CRD version
  - Report migration errors
- It might be possible that the manager is doing meaningful work while the upgrade is performed. 
  - This should be safe if everything works correctly, to be on the safe side it would be good to introduce an `idle` state 
    for the manager in which it does not perform any work and is safe to shutdown, update the CRDs, etc.

#### Failure cases
- Migration of CRDs fails while in process
  - Only when manually migrating CRs:
    - Restart of migration must be able to support a started migration
    - Detect an failed migration
    - Continue migrating CRDs
    - Start new version of manager
- New manager fails to start
  - Only available option here would be to roll back?
- Migration of prerequisites fails
  - Very case specific failure cases here, i.e. a namespace already exists, some permission is missing
  - These cases should all be checkable before the upgrade starts

## Infrastructure Needed
- Upgrade & Migration Test Harness
    - How old of a KUDO version do we test for upgrades
      - We will need some e2e-tests for upgrading KUDO, but we can't support every combination of upgrades. What is the lowest KUDO version that we test for upgrades?
      - N-2 (i.e. we provide tests that upgrade from KUDO 0.10.0 to 0.11.0 and 0.12.0)
      - Time-based (i.e. we provide tests that upgrade from the oldest KUDO version from 6 months ago)
      - Baseline (we keep a single KUDO version (i.e. 0.10.0) as baseline and keep tests for updating to the latest KUDO version)

## Resources
- [CRD versioning](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definition-versioning/)
- [Kube Storage Version Migrator](https://github.com/kubernetes-sigs/kube-storage-version-migrator)
- [K8s API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
- [K8s API Change Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md)


## Implementation History
- 2019/11/05 - Initial draft. (@aneumann82)
- 2020/01/07 - Cleanup, clarifications
- 2020/09/22 - Rework
