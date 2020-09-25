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
last-updated: 2020-09-24
status: provisional
---

# KUDO instance readiness

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)

## Summary

KUDO helps people implement their operators and it's focus is day 2 operations. Part of day 2 is also monitoring your workload readiness after deployment. To help with that, KUDO will expose readiness computed as a heuristic based on readiness of the underlying resources. In the first iteration, readiness will be just a simple heuristic computed from Pods, StatefulSets, Deployments, ReplicaSets, DaemonSets and Services (let's call them *readiness phase 1 resources*).

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
