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
last-updated: 2020-01-24
status: provisional
---

# cluster-resources-for-crds

To enable the creation use of certain CRDs, some cluster scoped objects may need to be created.  This should not be part of the creation of a particular CRD instantiation since the deletion of that instance would remove the dependency from all objects. Allowing an Operator or OperatorVersion to define a set of Cluster objects that are present to support the creation and management of CRDs would circumvent the CRD from having to create and manage the object.

## Table of Contents

* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [User Stories](#user-stories)
      * [Story 1](#story-1)
      * [Story 2](#story-2)
      * [Story 3](#story-3)
      * [Story 4](#story-4)
    * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
      * [Instance Status update](#instance-status-update)
      * [Object Owners](#object-owners)
      * [Uninstalls and Object Cleanup](#uninstalls-and-object-cleanup)
      * [Specifications of Operator](#specifications-of-operator)
    * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)
* [Drawbacks](#drawbacks-optional)


## Summary

To enable the creation use of certain CRDs, some cluster scoped objects, or common objects, may need to be created.  This should not be part of the creation of a particular CRD instantiation since the deletion of that instance would remove the dependency from all objects. Allowing an Operator or OperatorVersion to define a set of Cluster objects that are present to support the creation and management of CRDs would circumvent the CRD from having to create and manage the object.

## Motivation

### Goals

* Allow the creation of common Kubernetes components in support of running a CRD.

### Non-Goals

* A mechanism for CRDs to create cluster scoped objects.

## Proposal

### User Stories


#### Story 1

A database Operator may want a Restic server deployed as part of the Operator to provide a central location for storing backups. The capability here would define a namespace, service and deployment that provides Restic to the instance of the Operator to use by default.


#### Story 2

An Operator could leverage a MutatingWebHook for modifying pods deployments based on Node metadata (e.g. use a different image for nodes configured with GPU, PMEM, etc)


#### Story 3

The creation of a CRD should be controlled by a ClusterRole that defines permissions on who can create instances of the CRD.  These should probably be created

#### Story 4

OperatorVersions require the existence of CRDs that are not controlled by Kudo (e.g. ETCD Operator) and require those to be installed when OV is enabled.

### Implementation Details/Notes/Constraints

The installation of an Operator Version can run a plan specified by the Operator Version spec, when the `installPlan` value is specified in `operator.yaml`. The value shall match a defined `plan` in the `plans` spec in the OperatorVersion. As with the Instance Status properties, the OperatorVersion will contain

```golang
	PlanStatus       map[string]PlanStatus `json:"planStatus,omitempty"`
	AggregatedStatus AggregatedStatus      `json:"aggregatedStatus,omitempty"`
```

to contain the status information on the execution of the `installPlan`. The runtime logic for executing the plan on OperatorVersion install will be the same core logic that is used in the Instance plan execution.

#### Instance Status update

If Instances of an OperatorVersion are created before the Install plan of the OperatorVersion is successful, the Instance's will remain in a `PENDING` status until the OperatorVersion is successful.

#### Object Owners

The objects created by the `installPlan` may exist on the cluster before the OperatorVersion is installed. This may be a result of another OperatorVersion requiring the same objects, or the objects being managed externally to KUDO. If so, the OperatorVersion will be appended to the `ownerReferences`. If the object does not exist, then KUDO will create the object with the OperatorVersion as the `ownerReference`, and additionally add an annotation to the object that it was created by KUDO: `kudo.dev/managed: true`. This will be used during cleanup to identify whether the object was created by KUDO or not.

If the object already exists, but does not match the spec provided by the `installPlan`, the `installPlan` will fatally error and the OperatorVersion's status will be set to `FatalError`. We will not support updating/changing existing objects as part of an install since this may change how components not managed by this OperatorVersion would function.

#### Uninstalls and Object Cleanup

When an OperatorVersion is uninstalled, KUDO will remove the OperatorVersion from the `ownerReference`. If there are no more `ownerReferences`, it will look for the `kudo.dev/managed: true` annotation to identify if this object is no longer needed on the cluster. If that annotation is there, it will remove the object. If it is not there, then the object must have existed before installing any OperatorVersions and is managed by a component outside of KUDO, and will remain on the cluster.

#### Specifications of Operator

### Risks and Mitigations

## Graduation Criteria

The MySQL Operator could be modified to deploy a central repo for backups and leverage those in each instance of MySQL. Spark Operator will be able to install the required CRDs on OperatorVersion installataion that can be used by instances of the Spark Operator

## Implementation History

* Initial KEP - 20190218
* Proposal for Implementation - 20200124

## Drawbacks

* More complicated Operator/OperatorVersion specs
* Implications of OperatorVersion installation making MORE cluster level changes than just a CRD.
* Security around leveraging the common component.  If deploying minio for backups, how do we ensure Instances don't restore someone else's data, or overwrite someone else's backup?
* Shared Components: What happens when two different Operators require the same common object?
