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
      * [Proposal](#proposal)
         * [KUDO Prerequisites](#kudo-prerequisites)
         * [KUDO Manager](#kudo-manager)
         * [CRDs](#crds)
         * [KUDO control](#kudo-control)
         * [User Stories](#user-stories)
            * [Story 1](#story-1)
            * [Story 2](#story-2)
         * [Implementation Details/Notes/Constraints TODO [optional]](#implementation-detailsnotesconstraints-todo-optional)
         * [Risks and Mitigations TODO](#risks-and-mitigations-todo)
      * [Graduation Criteria TODO](#graduation-criteria-todo)
      * [Implementation History TODO](#implementation-history-todo)
      * [Drawbacks TODO [optional, TODO]](#drawbacks-todo-optional-todo)
      * [Alternatives TODO [optional]](#alternatives-todo-optional)
      * [Infrastructure Needed TODO [optional]](#infrastructure-needed-todo-optional)

<!-- Added by: aneumann, at: Thu Nov 28 16:03:05 CET 2019 -->

<!--te-->

## Summary

TBD

## Motivation

We strive for quick and regular updates of KUDO. We need a process for upgrading all the moving parts of KUDO, and how 
the different parts interact, and what kind of compatibility we want to provide.

### Goals

- How updates of KUDO are executed
  - Updates of CRDs
  - Updates of the KUDO manager
  - Updates of prerequisites (Namespaces, RoleBindings, ServiceAccounts, etc.)
- Interoperability
  - How and if multiple versions of CRDs are maintained
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
- Do we want to support downgrades? 
- Split versions between KUDO manager and KUDO CLI?

## Proposal

### KUDO Prerequisites
Expected update frequency: Low
Versioned: No, but closely tied to KUDO manager

The KUDO manager has a set of prerequisites that need to be present for the manager to run successfully. They are
the least likely to change, but probably the most specific. If we make changes here, we need to implement custom
migration code.

- Current prereqs
  - Namespace
  - Service Account
  - Role Bindings
  - Secret

#### Proposal for update process 
Integrated into `kudo init --upgrade`

Write specific migration code that targets a KUDO manager version range and executes manual migration steps. 

Each migration should have a validate-step that checks if the migration is possible.

**Alternative update process**
- Can we just delete them all and reinstall them? Probably not

### KUDO Manager
Expected update frequency: High
Versioned: Yes

The KUDO Manager is defined by an image version in a deployment set. To update, the deployment must be updated. The 
manager is closely tied to the CRDs, but not to the CLI. When CRDs are updated, the Manager will most likely also
need to be updated. 

#### Proposal for update process
Integrated into `kudo init --upgrade`

- Use semantic versioning for the manager binary
  - ToBeDiscussed: Use different versioning for Manager and CLI?
- The manager needs to be shut down before updating the CRDs
  - As soon as the new CRD is installed and marked as "stored"
  - Without WebHook Conversion, even if the new CRD version is "served", the old content is returned without any conversion
- Question: Do we need to ensure that the manager is not doing meaningful work at the moment, or can we just update the deployment?

### CRDs
Expected update frequency: Medium
Versioned: Yes, with a CRD-Version

The CRDs are used to store installed operators and running instances. New features will regularly require us to add new
fields. 
- Existing CRs need to be migrated to new versions
- K8s CRD support
  - Which K8s version do we want/need to support?
    - MultiVersion is supported since 1.11 (manual conversion)
    - WebHook conversion since 1.16
  - WebHook conversion would allow us to transparently switch to a new CRD version without manually migrating all existing CRs
- CRD versioning
  - Currently we have one global version for all CRDs
  - Alternatively, we could have a one version per CRD
  - Single version should be ok, in the worst case we have empty migrations
  - One version for each CRD would require complex compatibility matrix

#### Proposal for update process
Integrated into `kudo init --upgrade`

To support older K8s versions, no WebHook conversion is used. We only serve a single CRD-Version and migrate existing CRs in the update process

### KUDO CLI
Expected update frequency: High

KUDO CLI is the command line tool to manage KUDO. It will be often updated to add new features and fix bugs. It needs
to be in sync with the installed CRDs, as it's writing them directly with the K8s extension API.

- Do we allow an older KUDO CLI to be used with a newer KUDO installation?

#### Proposal for update process
User has to download newest KUDO version.

- CLI will have to support multiple CRD versions
  - At least one old and one new version to migrate CRDs
  - Possibly more than one to support older KUDO installations
  - Features not supported by a CRD version must be feature gated
  - We need to decide how long a specific CRD version is supported by CLI
- CLI must support the exact CRD version that is installed in a cluster. It will not be possible to use an old KUDO CLI on a newer cluster
  - CLI updates should be easy, therefore no need to introduce additional complexity here
  - CLI should support multiple CRDs anyway
  - With WebHook conversion the K8s cluster could support multiple CRDs, but that's out of scope for this KEP

## Updating KUDO installation

The update of a KUDO installation is triggered by  `kudo init --upgrade`

### Upgrade Steps
- Pre-Update-Verification
  - Detect if permissions to modify prerequisites are available
  - Verify old CRDs can be read by new KUDO version
  - Verify all installed operators are supported by new KUDO version
  - User can abort here
- Shutdown old manager version
- Install new CRDs
- Migrate all existing CRs to new format
  - Use Storage Version Migrator
  - or
  - Write custom code to migrate stored CRs
- Deploy new manager version

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

## Alternatives
A (possible future) alternative to updating the KUDO manager and migrating the CRDs is to use WebHook conversions. This would allow a no-downtime update, but require K8s 1.16 as minimum version.

- Install new Conversion WebHook
- Install new CRD version
  - Old CRD version is stored, and both versions can be read and written(?)
- Install new KUDO manager 
  - From now on, the new KUDO manager can work through WebHook Conversion with the new CRD format
- Mark new CRD version as "stored"
- At some point in the future (maybe a release later):
   - Switch the "served" flag on old CRD version to false
   - Disable migration code in Conversion WebHook

This will require a new KEP to workout the details.

## Infrastructure Needed
- Upgrade & Migration Test Harness

## Resources
- [CRD versioning](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definition-versioning/)
- [Kube Storage Version Migrator](https://github.com/kubernetes-sigs/kube-storage-version-migrator)

## Implementation History
- 2019/11/05 - Initial draft. (@aneumann82)