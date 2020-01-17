---
kep-number: 5
title: Cluster Resources for CRDs
short-description: Cluster Resources for CRDs
authors:
  - "@runyontr"
owners:
  - "@runyontr"
editor: TBD
creation-date: 2019-02-18
last-updated: 2019-02-19
status: provisional
---

# cluster-resources-for-crds

In order to enable the creation use of certain CRDs, some cluster scoped objects may need to be created.  This should not be
part of the creation of a particular CRD instantiation since the deletion of that instance would remove the
dependency from all objects. Allowing an Operator or OperatorVersion to define a set of Cluster objects that are present
to support the creation and management of CRDs would circumvent the CRD from having to create and manage the object.


## Table of Contents


* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [User Stories [optional]](#user-stories-optional)
      * [Story 1](#story-1)
      * [Story 2](#story-2)
    * [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)
* [Drawbacks [optional]](#drawbacks-optional)
* [Alternatives [optional]](#alternatives-optional)
* [Infrastructure Needed [optional]](#infrastructure-needed-optional)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Summary

In order to enable the creation use of certain CRDs, some cluster scoped objects, or common objects, may need to be created.  This should not be
part of the creation of a particular CRD instantiation since the deletion of that instance would remove the
dependency from all objects. Allowing an Operator or OperatorVersion to define a set of Cluster objects that are present
to support the creation and management of CRDs would circumvent the CRD from having to create and manage the object.

## Motivation

### Goals

* Allow the creation of common Kubernetes components in support of running a CRD.

### Non-Goals

* A mechanism for CRDs to create cluster scoped objects.

## Proposal

### User Stories [optional]


#### Story 1

A database Operator may want a Restic server deployed as part of the Operator to provide a central location for storing backups.
The capability here would define a namespace, service and deployment that provides Restic to the instance of the Operator to use
by default.


#### Story 2

An Operator could leverage a MutatingWebHook for modifying pods deployments based on Node metadata (e.g. use a different image for nodes configured with GPU, PMEM, etc)


#### Story 3

The creation of a CRD should be controlled by a ClusterRole that defines permissions on who can create instances of the CRD.  These
should probably be created

#### Story 4

OperatorVersions require the existance of CRDs that are not controlled by Kudo (e.g. ETCD Operator) and require those to be installed when FV is enabled.

### Implementation Details/Notes/Constraints [optional]



### Risks and Mitigations



## Graduation Criteria

The MySQL Operator could be modified to deploy a central repo for backups and leverage those in each instance of MySQL

## Implementation History

* Initial KEP - 20190218

## Drawbacks [optional]

* More complicated Operator/OperatorVersion specs
* Implications of OperatorVersion installation making MORE cluster level changes than just a CRD.
* Security around leveraging the common component.  If deploying minio for backups, how do we ensure Instances don't restore someone else's data, or overwrite someone else's backup?
* Shared Components: What happens when two different Operators require the same common object?
