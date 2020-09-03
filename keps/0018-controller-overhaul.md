---
kep-number: 18
title: Controller redesign
short-desc: Refactoring of KUDO controller
authors:
  - "@alenkacz"
owners:
  - "@alenkacz"
creation-date: 2019-09-16
last-updated: 2019-09-16
status: implementable
---

# Controller redesign

## Table of Contents

* [Controller redesign](#controller-redesign)
      * [Table of Contents](#table-of-contents)
      * [Summary](#summary)
      * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals)
      * [Proposal](#proposal)
         * [Instance CRD changes](#instance-crd-changes)
         * [Admission webhook](#admission-webhook)
         * [Implementation details](#implementation-detailsnotesconstraints)
         * [Risks and mitigations](#risks-and-mitigations)
      * [Graduation Criteria](#graduation-criteria)

## Summary

The way KUDO controllers are currently implemented has several flaws:
1. When KUDO manager is down (or restarted) we might miss an update on a CRD meaning we won’t execute the plan that was supposed to run or we execute an incorrect one ([issue](https://github.com/kudobuilder/kudo/issues/422))
2. Multiple plan can be seemingly in progress leading to misleading status communicated ([issue](https://github.com/kudobuilder/kudo/issues/628))
3. No way to ensure atomicity on plan execution (given the current CRD design where information is spread across several CRDs)
4. Very low test coverage and overall confidence in the code

To address all of the above points we’re proposing a new design of our CRDs. We’re going to get rid of PlanExecution CRD altogether and merge the execution status into the Instance’s status subresource. At the same time, we’re going to keep last known state of instance inside status subresource as well, which will allow us to properly identify plans that are supposed to run after the user submits a change to the Instance spec.

## Motivation

### Goals

- better understandable and tested controller code
- ensuring the correct plan is executed in case the KUDO controller is not running for some time
- atomicity on plan execution level - do not allow to run a plan when another plan is in progress
- be able to see current plan execution status directly in the Instance

### Non-Goals

- providing longer history of plan executions that just the last one run
- have manual plan execution feature (there are several approaches to how to do this, I think it deserves a separate KEP)

## Proposal

The proposed design that will solve the above-defined issues with controllers and CRDs is to get rid of PlanExecution CRD altogether. Plan status will basically become instance status (it’s the case right now as well anyway, we just need to replicate those values across two places). 

### Instance CRD changes

The instance CRD for instance with running deploy plan might then look as following.

```yaml
 Instance
  metadata:
    annotations:
      "kudo.dev/last-applied-instance-state": "{ ... }" # InstanceSpec type serialized as json into string
  spec:
    operatorVersion: operator-123
    parameters:
      - REPLICAS: 4
  status:
    planStatus: # (for every plan defined in OV, there will be one field)
      - deploy:
          status: IN_PROGRESS
          executionName: deploy-1478631057
          lastFinishedRun: 1568642057
          overview:
            phases:
              - name: deploy
                status: IN_PROGRESS
                steps:
                  - name: everything
                    status: IN_PROGRESS
      - upgrade: null # (never run)
    aggregatedStatus:
      - status: IN_PROGRESS
        activePlan: deploy-1478631057 # (will be null if no plan is run right now)
```
`planStatus` is a property that basically replaces the current PlanExecution CRD - it reports on the status of the plans that are run right now or last runs of all the plans available for that operator version. This is also what `kudo plan history` and `kudo plan status` will query to get overview of the plans.

`kudo.dev/last-applied-instance-state` annotation is persisting the state of the instance from the previous successful deploy/upgrade/update plan finished. We need this to be able to solve flaw n.1 described in the summary. This gets updated after a plan succeeds.

`status.aggregatedStatus.status` is just a simple way how to query the overall state of the instance (if any plan is running on it or not). Could be used by user as well as e.g. admission webhook to figure out if any changes to the CRD are allowed.

### Admission webhook

Part of the solution (addressing problem n.3 from the summary - ensuring atomicity) is an admission webhook. This one will guard the Instance CRD and will disallow any changes in a spec if plan is running. Admission webhook in kubernetes world is the only way how to prevent resource from being updated. All following filters (like controller predicates) are called AFTER the change was applied so it's too late to validate.

Although admission webook addresses one of the problems outlined in this KEP it's considered a stretch goal and can be delivered in a following release (should NOT be part of the initial refactoring).

### Implementation Details/Notes/Constraints

- don't use predicates to process any kind of business logic (these don't run in worker pool and should be just simple functions returning bool)
- overall try to aim to use as much best practices in using controller-runtime as possible
- implementation of this will be a breaking change meaning that KUDO on existing clusters will have to be reinstalled to work (CRDs dropped and recreated)
- temporarily we won't be able to execute manual jobs until we agree on a design there (none of the current operators use it anyway so should be fine for one release)

### Risks and mitigations

This is a pretty big refactoring (touching controllers, CLI, tests) that will likely end with a big diff. The goal to limit that was done by doing some ground work in [PR #759](https://github.com/kudobuilder/kudo/pull/759). With code-walkthrough done for team members as well as with our current integration test code coverage I feel pretty confident when landing this, we won't end up in a worse situation as we are right now (although some of the integration tests have to be changed as well - especially those working with PlanExecution object).

To make the probability of landing something broken even smaller, try to move in small steps and involve other team members as soon as possible.

Admission webhook is an easily separate part of the implementation that should land separately and can even be worked on in parallel.

## Graduation Criteria

When current integration tests passes and the new tests passes as well (e.g. I introduced this test [PR #718](https://github.com/kudobuilder/kudo/pull/718) but it's failing on main, that should not be the case when we land this feature).
