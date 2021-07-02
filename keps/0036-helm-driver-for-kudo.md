---
kep-number: 36
short-desc: HELM driver for KUDO
title: HELM driver for KUDO
authors:
  - "@rishabh96b"
owners:
  - TBD
editor: TBD
creation-date: 2021-03-11
last-updated: 2021-03-11
status: provisional
---

# helm driver for kudo proposal

## Table of Contents

* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
* [Implementation History](#implementation-history)
* [Drawbacks [optional]](#drawbacks-optional)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Summary

KUDO enables Operator developers to create Operators and expose features of the underlying applications. KUDO helps in the lifecycle management of the underlying applications by enabling develpers to create custom tasks and plans.
The goal of this KEP is to facilitate the helm driver for kudo so that developers can define custom tasks and plans for their applications packaged in helm to provide developers/admins can achieve better application lifecycle management.

## Motivation

Application operators often want to perform maintenance on an application that is unique to that application. KUDO's ability to define custom operations as part of its definition makes common actions less error prone. This functionality is missing in helm forcing develpers/admins to rely on manual actions for performing administrative tasks. As KUDO possess this powerful capability, a helm driver would aim to fill this gap in helm based deployments.

### Goals

- capability to extend helm packaged applicatins with kudo to write custom plans/tasks.

### Non-Goals



## Proposal

- TBD


## Implementation History

## Drawbacks

- Could require significant development efforts.

