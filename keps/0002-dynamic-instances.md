---
kep-number: 2
title: Dynamic Instances
authors:
  - "@gerred"
owners:
  - "@gerred"
editor: TBD
creation-date: 2019-02-14
last-updated: 2019-05-10
status: implementable
---

# operator-specific Dynamic Custom Resource Definitions

## Table of Contents

- [operator-specific Dynamic Custom Resource Definitions](#operator-specific-dynamic-custom-resource-definitions)
  - [Table of Contents](#table-of-contents)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Implementation History](#implementation-history)

## Summary

This KEP aims to make the end user experience for operators more specific to the business domain they represent. By implementing dynamic instances, operators will be able to represent their components in a declarative way, specific to their operator. Additionally, it enables operator developers to provide day 2 operations tasks as CRDs, complete with their own plans and tasks.

## Motivation

Currently, the interface for using operators in KUDO is generic. Users create an `Instance` CRD with specs. Operator developers and users expect to be able to use contextual business objects for their operators instead of generic objects. This enables a more focused experience for users of KEP.

The goal of this KEP is to improve the end user UX through dynamic instances. Other than the ability to specfiy CRDs, and adjusting existing operator development instances to accomodate this change, it is not the goal of this KEP to change the operator development UX.

### Goals

- Create a mechanism for operator developers to specify an instance CRD
- Enable management for custom resources based on dynamic instances. Deploying an operator specific custom resource should deploy a plan as `Instance` was able to before.

### Non-Goals

- Change the operator developer UX for templates, parameters, tasks, and plans.

## Proposal

Currently, there are issues with CRD storage in Kubernetes that lead to watches not cancelling when a CRD is uninstalled. The related fix for this in Kubernetes will be part of https://github.com/kubernetes/kubernetes/issues/68508. Implementing this KEP will be divided into 5 steps, based on the state of [Kubernetes CRDs achieving General Availability status](https://github.com/kubernetes/kubernetes/issues/58682) and https://github.com/kubernetes/kubernetes/issues/68508:

1. Continue using Instance CRD

2. Change the Instance CRD to a struct that does not change when migrating between Instance and CRD. This struct would be similar to:

```go
type Instance struct{
    metadata metav1.ObjectMeta
    apiVersion string
    spec struct{
        operator *string
        Version string
        Arguments map[string]string
    }    
}

```

3. Add a package for the CLI to use that abstracts the details of generating out the CRD on install. The CLI will use this package.

After the Kubernetes changes land:

4. Write an Instance to Dynamic Instance migrator. This will statically convert Instance templates to their corresponding Dynamic Instance CRD for use by operators.
5. Add a Mutating Admission Controller that converts Instance CRDs to their corresponding Dynamic Instance. This enables GitOps users to continue operating as they do today without breaking their pipelines.

### Risks and Mitigations

The primary risk of this is that a fix for graceful deletion of CRDs never lands, or takes too long to land. In that instance, we would consider building a KUDO api-server as an aggregated or standalone API.

## Implementation History

- `Summary`, `Motivation`, and `Goals` being merged.
- Add Proposal and Risks
