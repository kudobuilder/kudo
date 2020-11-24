---
kep-number: 34
short-desc: Provide a way to monitor readiness of KUDO instance post-deployment
title: KUDO instance readiness
authors:
  - "@alenkacz"
owners:
  - "@alenkacz"
editor: TBD
creation-date: 2020-09-24
last-updated: 2020-11-19
status: implemented
---

# KUDO instance readiness

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)

## Summary

KUDO helps people implement their operators and it's focus is day 2 operations. Part of day 2 is also monitoring your workload readiness after deployment. To help with that, KUDO will expose readiness computed as a heuristic based on readiness of the underlying resources. In the first iteration, readiness will be just a simple heuristic computed from Pods, StatefulSets, Deployments, ReplicaSets and DaemonSets (let's call them *readiness phase 1 resources*).

## Motivation

Added readiness monitoring for `Instance` CR will help people answer question "is my operator running and ready at this point in time" (considering all the available information exposed by k8s resource) without querying all underlying resources. KUDO will expose this heuristic as part of `Status` field for everyone to query. This could be used as a signal for a monitoring tool.

The idea here is very similar to the relation between `Deployment` and `Pod` core k8s types. Pods contain very low-level information about their readiness and state they are in while `Deployment` tries to compute an aggregated and higher-level state from all the underlying owned resources. The same goal now applies to `Instance`.

### Goals

Expose readiness information in `Status` field of `Instance`
Compute readiness by evaluating readiness of *readiness phase 1 resources*

### Non-Goals

Drift detection (detecting that resource was deleted or changed manually)
Including other types of resources than *readiness phase 1 resources*
Determining if the underlying application is functional
Determining if the underlying application is reachable

## Proposal

Readiness of an `Instance` will be communicated via a newly introduced `Status.Conditions` field. This field as a convention is an array of items that will [conform the schema](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L1367) recommended by k8s api machinery. The `Type` of the newly added condition will be `Ready`. Condition is supposed to have these [three values](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L1288-L1298) which will have the following meaning:
- True - the last observation of readiness on this instance is that it’s ready
- False - the last observation of readiness on this instance is that it’s NOT ready
- Unknown - deploy/upgrade/update plan is running or last plan execution of these plans ended with FATAL_ERROR

Example of Instance status being ready:
```
apiVersion: kudo.dev/v1beta1
kind: Instance
Status:
  Conditions:
    - type: Ready                     
      status: "True"
      lastTransitionTime: 2018-01-01T00:00:00Z
```

Example of Instance status being NOT ready:
```
apiVersion: kudo.dev/v1beta1
kind: Instance
Status:
  Conditions:
    - type: Ready                     
      status: "False"
      lastTransitionTime: 2018-01-01T00:00:00Z
      Reason: ResourceNotReady
      Message: Deployment ‘xxx’ is not ready, Deployment ‘yyy’ is not ready
```

Example of status being Unknown for running plan:
```
apiVersion: kudo.dev/v1beta1
kind: Instance
Status:
  Conditions:
    - type: Ready                     
      status: "Unknown"
      lastTransitionTime: 2018-01-01T00:00:00Z
      Reason: PlanRunning
      Message: Plan 'deploy' is currently running
```

Ready is expected to be an oscillating condition and it will indicate the resources owned by the instance were believed to be ready at the time it was last verified. There’s an expectation that the value won’t be stalled for more than 1 minute (at least one minute after the state changes, status should be reflecting it).

Unknown state will be set whenever deploy/upgrade/update plan is running - this is because a plan run should be an atomic operation going from stable state to another stable state, evaluating health of all resources involved in the plan as part of the execution. To document why having an unknown state for plan running is important let's think of the following example: plan has a pre-install phase and install phase. While pre-install is running, no deployments exist yet, but that would mean that we would mark that instance as `Ready:true` even though not all deployments were applied yet. The same goes for FATAL_ERROR state - in this case we simply cannot tell at which phase the execution was stopped so we cannot really guarantee that the computed `Ready:true` signal would be meaningful and user can consider that installation to be ready. 

The reason for using Conditions field rather than introducing a separate field is mainly that it’s starting to be established as a go-to pattern in kubernetes world. Also conditions have a standard set of fields attached with metadata that are useful for our case (namely Reason, Message). Having a ‘Ready’ condition is also [encouraged by sig-architecture](https://github.com/kubernetes/community/pull/4521/files). (excerpt from that document: *Although they are not a consistent standard, the `Ready` and `Succeeded` condition types may be used by API designers for long-running and bounded-execution objects, respectively.*)

### Implementation Details/Notes/Constraints

Setting `Ready` condition will be a responsibility of existing  `instance_controller`. It will be watching for all the types in *readiness phase 1 resources* (it’s already watching most of those already so there’s really no additional big performance penalty) and trigger reconcile for owned Instance.

Controller will have to pull all the *readiness phase 1 resources* (this means additional N requests to API server where N is number of resource types) while filtering for labels `heritage: kudo` and `kudo.dev/instance=<instance-name>`. 

From all these resources it would compute readiness based on `Condition: Ready`, `Condition: Available` or other fields based on the convention of that particular type.

For operators installing their app via e.g. `Deployment`, there’s no need at this point to also check the Status of the underlying Pods because Deployment `Status` already mirrors the readiness of the underlying Pods. This is true to all higher-level types like StatefulSet etc.

To mitigate situation when a plan adds a resource we don't want to include in `Ready` condition, it's possible to opt-out from this by adding annotation `kudo.dev/skip-for-readiness-computation:true`

### Risks and Mitigations

On a big cluster with a huge number of Pods and Deployments, it’s possible that a controller might have a scaling issues because of number of items it need to process.

This is mitigated by a fact that inside event handlers we’re filtering only for events that belong to KUDO which should limit the scope of events to only a very few (the expectation is that majority of the Deployments does not come from KUDO).

Also this feature is not making this situation any worse as it’s right now because KUDO controller already watches for Pods and Deployments, so we’re not introducing new overhead. That said the controller will have to perform much more work in times where it was just "idling" before - because it was working only when plan was run, otherwise the reconcile ended right after it started. This could pose problem on bigger clusters with many KUDO operators and could be mitigated by running KUDO controller with multiple workers (right now 1 worker is enough on most installations we're aware of).

### Future work

To support even more complex cases and operator, we should think about adding possibility to specify `kudo.dev/readiness-strategy:selected` which will switch readiness computation to OPT-IN mode. With this one set, only resources with `kudo.dev/allow-readiness-computation:true` will be considered for readiness computation.

A plan is also to add more Conditions like `InProgress` capturing currently running plan.
