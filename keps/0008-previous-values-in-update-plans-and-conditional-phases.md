---
kep-number: 8
title: Previous values in update plans and conditional phases
authors:
  - "@djannot"
owners:
  - TBD
editor: TBD
creation-date: 2019-04-01
status: provisional
---

# Previous values in update plans and conditional phases

## Table of Contents

  * [Table of Contents](#table-of-contents)
  * [Summary](#summary)
  * [Motivation](#motivation)

Created by [gh-md-toc](https://github.com/ekalinin/github-markdown-toc)

## Summary

In order to develop the logic behind an `update` `Plan`, you generally need to know what were the previous values.

For example, if the number of replicas of a `StatefulSet` is updated, you ofter need to add a new `Phase` to reconfigure your cluster.

But currently, you don't know what has changed. You don't know if the number of replicas has changed and if it has changed you don't know what was the previous value.

We would need a way to access the previous values, but also probably to be able to define `Phases` that need to be run based on conditions.

## Motivation

Knowing what were the previous values will help someone who develop a KUDO framework to define the logic that needs to take place when an update occurs.
