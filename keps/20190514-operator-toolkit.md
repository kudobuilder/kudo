---
kep-number: 10
title: KUDO Operator Toolkit
authors:
  - "@kensipe"
  - "@gerred"  
owners:
  - "@kensipe"
  - "@gerred"
editor: "@kensipe"
creation-date: 2019-10-14
last-updated: 2019-10-14
status: provisional
see-also:
  - KEP-0002
---

# KUDO Operator Toolkit

## Table of Contents

* [KUDO Operator Toolkit](#kudo-operator-toolkit)
     * [Table of Contents](#table-of-contents)
     * [Summary](#summary)
     * [Motivation](#motivation)
     * [Proposal](#proposal)
        * [Goals](#goals)
        * [Non-Goals](#non-goals)

## Summary

Drive KUDO to have a set of tools, frameworks and specifications for creating operators.


## Motivation

KUDO provides a way to reduce the amount of code necessary for the creation of an operator for Kubernetes. The current implementation of KUDO requires a significant amount of YAML to define an operator.   This YAML is verbose, is not reusable, and requires [kustomize](https://github.com/kubernetes-sigs/kustomize) and [mustache](https://mustache.github.io/).  We believe we can significantly improve the operator developers experience through the creation of a toolkit.


## Proposal

### Goals

The toolkit must provide the ability for:

* conditional inclusions
* defining default values
* replacement of values
* removal of keys and values

### Non-Goals

* Any new KUDO features will not be covered by this KEP
