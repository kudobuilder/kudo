---
kep-number: 21
title: Upgrades of KUDO installation
authors:
  - "@aneumann82"
owners:
  - "@aneumann82"
editor: TBD
creation-date: 2019-11-28
last-updated: 2019-12-02
status: provisional
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
      * [Open Questions](#open-questions)
      * [Proposal](#proposal)
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
         * [Risks and Mitigations TODO](#risks-and-mitigations-todo)
            * [Failure cases](#failure-cases)
      * [Graduation Criteria TODO](#graduation-criteria-todo)
      * [Drawbacks TODO [optional]](#drawbacks-todo-optional)
      * [Alternatives](#alternatives)
      * [Infrastructure Needed](#infrastructure-needed)
      * [Resources](#resources)
      * [Implementation History](#implementation-history)

<!-- Added by: aneumann, at: Mon Dec  2 11:34:32 CET 2019 -->

<!--te-->

## Summary

TBD

## Motivation

We strive for quick and regular updates of KUDO. We need a process for upgrading all the moving parts of KUDO, and how 
the different parts interact, and what kind of compatibility we want to provide.

### Goals

- Determine how updates of KUDO are executed
  - Updates of prerequisites (Namespaces, RoleBindings, ServiceAccounts, etc.)
  - Updates of CRDs (Versioning, migration between versions)
  - Updates of the KUDO manager
- Interoperability
  - If and how many multiple versions of CRDs are maintained
  - How a version of KUDO CLI work with older and newer CRD versions
- How to handle operators that are not supported by a new KUDO version

### Non-Goals

- Updates/Upgrades of operators and operator versions itself
- Versioning of KUDO itself
- Multi-Tenancy, i.e. running multiple KUDO installations on the same cluster

### Current State

At the moment, KUDO does not provide any migration capabilities and needs a clean installation to use a new version.

## Open Questions
- Lowest supported K8s version
  - This determines if we use conversion webhooks or not
- Do we want to support downgrades? 
- What is the lowest KUDO version that will support updates?
  - Do we support update from KUDO 0.8.0 (which does not have any update support) to a higher version?
  - Do we take the freedom to require a fresh install for the first KUDO version with an implementation of this KEP?
- How old of a KUDO version to we test for upgrades
  - We will need some e2e-tests for upgrading KUDO, but we can't support every combination of upgrades. What is the lowest KUDO version that we test for upgrades?
  - N-2 (i.e. we provide tests that upgrade from KUDO 0.10.0 to 0.11.0 and 0.12.0)
  - Time-based (i.e. we provide tests that upgrade from the oldest KUDO version from 6 months ago)
  - Baseline (we keep a single KUDO version (i.e. 0.10.0) as baseline and keep tests for updating to the latest KUDO version)

## Proposal

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

Two possible implementations

1. Write specific migration code that targets a KUDO manager version range and executes manual migration steps. 
    - Each migration should have a validate-step that checks if the migration is possible.
      - This might be problematic if multiple steps are to be executed - Can we validate a steps before the previous one is applied?
    - The update for prerequisites is tied to the version of KUDO manager:
      - There is a list of migrations:
        - MigrationTo0_9_0
        - MigrationTo0_10_0
        - MigrationTo0_11_0
    - KUDO CLI checks installed version of manager
    - Every migration step after the installed version is executed
    - Every migration step only migrates the prerequisites from the previous version to the marked version of the migration

2. Have only one setup/update version in KUDO
    - The setup/update contains a list of all prerequisites in correct order
    - Each prereq validates the current installed state, and verifies that it can install/update the current state to the expected state
    - Prereqs that are deleted in newer versions need to stay in the list of prerequisites

### KUDO Manager
Expected update frequency: High  
Versioned: Yes

The KUDO Manager is defined by an image version in a deployment set. To update, the deployment must be updated. The 
manager is closely tied to the CRDs, but not to the CLI. When CRDs are updated, the Manager will most likely also
need to be updated. 

#### Proposal for update process
Integrated into `kudo init --upgrade`

- Use semantic versioning for the manager binary
- As updates to the same CRD version should be backwards compatible, the manager can keep running while the CRD version is updated
- After CRD update we can deploy the new manager version 
- Question: Do we need to ensure that the manager is not doing meaningful work at the moment, or can we just update the deployment?

### CRDs
Expected update frequency: Medium  
Versioned: Yes, with a CRD-Version

The CRDs are used to store installed operators, running instances and other custom persistent data. New features will regularly require us to add new
fields or even new CRDs.

- Existing CRs need to be migrated to new versions
- K8s CRD support
  - WebHook conversion would allow us to transparently switch to a new CRD version without manually migrating all existing CRs
    - MultiVersion is supported since 1.11 (manual conversion)
    - WebHook conversion GA since 1.16 (1.13 alpha feature gate, 1.15 beta feature gate)
  - Manual CRD conversion would allow us to target a lower K8s version, but require code to migrate CRs in the upgrade process.
- CRD versioning
  - If we want to keep the same API change conventions as K8s, we will have a slower development pace and probably a lot of version changes
  - We *could* go with less strict conventions - if we ensure that the used KUDO CLI version is at least as high as the installed KUDO manager, 
  we could add new (optional) fields in the CRDs and be sure that the fields are not dropped when round tripping from the cluster to CLI and 
  back to the cluster.
  - If we have only the CRD version to rely on, we can add new optional fields, but have to accept the risk that an older version of the CLI
  silently drops the fields, as it doesn't know about them. (Correct me here if I'm wrong)
  - More breaking changes (removing fields, making fields required, renaming fields, etc. ) require a CRD version change

- CRDs may have a similar upgrade process as prerequisites:
  - Create/Update might be simple
  - We may have CRDs that are outdated and not used anymore and should be deleted  

#### Proposal for update process
Integrated into `kudo init --upgrade`

To be discussed.

### KUDO CLI
Expected update frequency: High

KUDO CLI is the command line tool to manage KUDO. It will be often updated to add new features and fix bugs. It needs
to be in sync with the installed CRDs, as it's writing them directly with the K8s extension API.

- Do we allow an older KUDO CLI to be used with a newer KUDO installation?
  - No. We would run into problems with the old KUDO CLI silently removing new fields from the CRDs
- Do we allow a newer KUDO CLI to be used with older KUDO installation?
  - Yes. If we want to prevent users having to install multiple KUDO versions, we need to support this. 
  - We need to decide how long and what version range of older KUDO installations we want to support.
  - Having to maintain multiple CRD versions inside KUDO CLI may be difficult. We would need to have checks on a new features if the feature is supported by an old KUDO installation

#### Proposal for update process
User has to download newest KUDO version, either manually or via `brew` or other means.

- CLI must be at at least the version of the installed KUDO manager. It will not be possible to use an old KUDO CLI on a newer cluster
- CLI updates should be easy, therefore no need to introduce additional complexity here

## Updating KUDO installation

The update of a KUDO installation is triggered by  `kudo init --upgrade`

### Upgrade Steps
- Pre-Update-Verification
  - Detect if permissions to modify prerequisites are available
  - Verify that all prerequisite upgrade steps could be executed
  - Verify old CRDs can be read by new KUDO version
  - Verify all installed operators are supported by new KUDO version
  - User can abort here
- Shutdown old manager version
- Install new CRDs
  - Set "served" flag on new CRD versions to "true"
- Migrate all existing CRs to new format
  - Use Storage Version Migrator
  - Write custom code to migrate stored CRs
  - If we use the CRD conversion webhooks, no work is required at this point
- Deploy new manager version
- If we use the CRD conversion webhooks, we need to set the new CRD version as "stored" and trigger an update of all existing CRs to migrate them
    - At some point in the future (maybe a release later):
       - Switch the "served" flag on old CRD version to false
       - Disable migration code in Conversion WebHook
   
## User Stories

#### Story 1

An Operator wants to upgrade KUDO to the latest version to utilize new features. All installed operators should continue
to work.

#### Story 2

An Operator manages two K8s clusters with different KUDO versions installed. How does he manage to control both in the
most easy way?

### Implementation Details/Notes/Constraints

- Base migration starts by comparing installed KUDO version vs. executing KUDO CLI version
  - Check if prereq migration part is required
  - Check if CRD migration part is required
  - Check if manager migration part is required
- Run pre-update verification for each part
- TODO

### Risks and Mitigations TODO

This operation will **need** a `--dry-run` option
- Do normal Pre-Update-Verification
- Read all existing CRs and run migration to new CRD version
  - Report migration errors

#### Failure cases
- Migration of CRDs fails while in process
  - Only when manually migrating CRs:
    - Restart of migration must be able to support a started migration
    - Detect an failed migration
    - Continue migrating CRDs
    - Start new version of manager
- New manager fails to start
  - Only available option here would be to downgrade?
- Migration of prerequisites fails
  - Very case specific failure cases here, i.e. a namespace already exists, some permission is missing

TODO: Security implications

## Graduation Criteria TODO

How will we know that this has succeeded?
Gathering user feedback is crucial for building high quality experiences and owners have the important responsibility of setting milestones for stability and completeness.
Hopefully the content previously contained in [umbrella issues][] will be tracked in the `Graduation Criteria` section.

[umbrella issues]: https://github.com/kubernetes/kubernetes/issues/42752

## Drawbacks TODO [optional]
Why should this KEP _not_ be implemented.

## Infrastructure Needed
- Upgrade & Migration Test Harness

## Resources
- [CRD versioning](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definition-versioning/)
- [Kube Storage Version Migrator](https://github.com/kubernetes-sigs/kube-storage-version-migrator)
- [K8s API Change Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md)


## Implementation History
- 2019/11/05 - Initial draft. (@aneumann82)



## Action Items (to be converted into issues/tasks)
- Extend `init` command to support `--upgrade`
- Create structure to support general handling for prerequisite upgrades (validate, create, update, delete)
- Create code to update manager deployment
- Implement Installed Operator check
  - Validate that all installed operators are compatible with new KUDO version
- Write e2e test harness for KUDO updates
- Write e2e tests that run KUDO CLI against old/different version of CRDs(?)
- (Write code to support feature flags based on installed CRDs/KUDO version?)
- (Implement WebHook Conversion Endpoint in Manager)
- (Add init code to install webhook conversion CR)
