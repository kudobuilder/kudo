---
kep-number: 31
title: Namespace Management
short-desc: Defining the way in which KUDO will work and manage namespaces
authors:
  - "@kensipe"
owners:
  - "@kensipe"
editor: @kensipe
creation-date: 2020-05-06
last-updated: 2020-05-13
status: provisional
---

# Namespace Management

## Table of Contents

 * [Summary](#summary)
 * [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
 * [Namespace Management in the Eco-System](#namespace-management-in-the-eco-system)
    * [Kubernetes Objects and Namespaces](#kubernetes-objects-and-namespaces)
    * [Helm](#helm)
    * [The Many Meanings of a Namespace](#the-many-meanings-of-a-namespace)
    * [Hierarchical Namespace Controller](#hierarchical-namespace-controller)
 * [Proposals](#proposals)
    * [Proposal: Single Namespace Support](#proposal-single-namespace-support)
    * [Proposal: Namespace Creation](#proposal-namespace-creation)
    * [Proposal: Multi-Namespace (static)](#proposal-multi-namespace-static)
    * [Proposal: Dependencies and Creating Namespaces](#proposal-dependencies-and-creating-namespaces)
    * [Proposal: Namespace Cleanup](#proposal-namespace-cleanup)
 * [Notes](#notes)
 * [Alternatives](#alternatives)
    * [Proposal: No namespace creation](#proposal-no-namespace-creation)

## Summary

This KEP aims to provide a definition on namespace management and provide guidance and constraints a KUDO operator may use namespaces.  This includes the expectations of creation, deletion, defining how an operator will install to a namespace and the ability of an operator to leverage more than one namespace.

## Motivation

Like many Kubernetes Objects, there is an expectation that a KUDO operator is organized by namespace.  This KEP defines expectations around namespace creation and use.

### Goals

In order to support a multi-namespace environment, it is necessary to support cluster-wide resources (including kudo instances) which is in part (or full) defined in [KEP-05](0005-cluster-resources-for-crds.md).  Based on this constraint, the goals are limited to the existing limitations in KUDO of having namespace only support for instances and not having cluster-wide resource support.

* Define if a namespace is a prerequisite to operator installation
* Define if a namespace is created by KUDO
* Define support for advanced namespaces (namespaces with metadata)
* Define if an operator can be installed into a new namespace

### Non-Goals

* multi-namespace support for an operator
* support for an operator to create or manage a namespace
* support for namespace deletion

## Namespace Management in the Eco-System

### Kubernetes Objects and Namespaces

When working kubernetes objects, be it Pods, ReplicaSets, Deployments, etc. that require a namespace, the responsbility falls on the user to create that namespace prior assigning the resource or it will fail.  Available is an [auto-provision admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#namespaceautoprovision) to auto-create namespaces. Users desiring auto-provisioning of namespaces should install this tool.  This webhook has the limitation of creating namespaces without extra metadata which defeats the value for use cases which need that metadata.

### Helm

Helm v1 and v2 provided an auto-provision capability.  The initial release of Helm v3 removed this capability and now requires users to create namespaces prior to installing a helm chart. Much detail is written in comments for [helm issue 6794](https://github.com/helm/helm/issues/6794).  Helm v3 [reintroduced the ability](https://github.com/helm/helm/issues/6794) to create a namespace through the use of `--create-namespace` flag.

### The Many Meanings of a Namespace

Challenages to auto-provision of namespaces are complicated by the many reasons / values for creating them which includes:

1. Logical separation of work (separating 1000s of pods into 10s of buckets for management)
1. Quota management
1. Security separation
1. Metadata (where labels and annotations provide meaning and value to controllers managing resources in a namespace)

Based on the last 3 reasons, it is common for an administrator to create namespaces to manage the other concerns associated with a namespace.  It would be challeging to manage a cluster where dozen or more namespaces are adhoc being created (or deleted) which may require a host of other configurations around quota and security.

### Hierarchical Namespace Controller

Currently in incubation, Kubernetes is introducing a Hierarchical Namespace Controller to more easily create and manage namespaces in the cluster, even if the user doesn't have cluster-level permission to create namespaces, which is something to keep in mind in the future.  See https://github.com/kubernetes-sigs/multi-tenancy/tree/master/incubator/hnc

## Proposals

Prior to making KEP `provisional` we will chose a namespace creation proposal to move forward with and move the 2nd proposal to alternative.  For now, we have 2 competing namespace creation proposals.  
In addition there is a proposal for the use of multiple namespaces.

### Proposal: Single Namespace Support

There is a desire to support multiple namespaces, however it requires the addition of cluster-wide resource support in KUDO.  Based on this and the desire to not break Kubernetes garbage collection, KUDO currently only supports 1 namespace.  Which results in the following explicit rules for operators until a cluster-wide support is provided:

* All manifests with metadata defining a namespace will be ignored or overridden.  The namespace that is being used by KUDO to install the operator is the only namespace that resources will be installed to.
* All namespace manifests (manifest of `"kind": "Namespace"`) will have no affect as they can not be used by the operator.
* All namespace manifests will be flagged by `package verify` as an error

### Proposal: Namespace Creation

KUDO should support the creation of a namespace if it does not exist.  Further KUDO will add a configuration for namespace such that at time of namespace creation it is created with the metadata provided. To this end a manifest file for namespace is to be supported in the template folder:

```
apiVersion: v1
kind: Namespace
metadata:
  labels:
    ca.istio.io/override: "true"
    istio-injection: enabled
    katib-metricscollector-injection: enabled
```

The template supports templates for labels and annotations, however the `name:` metadata is NOT allowed to be set or controlled through templating.   This is to prevent the "templating" of the namespace name such as `name: {.Namespace}-extended`. Other templating of labels and metadata is allowed and is the reason the file lives under the template folder. KUDO will add the `name:` for metadata when there is a need to create a namespace with metadata using the namespaceManifest file. The `operator.yaml` file will be extended to support `namespaceManifest` attribute.  

```
# operator.yaml
name: "kubeflow"
operatorVersion: "0.2.0"
kudoVersion: "0.10.1"
kubernetesVersion: 1.15.0
appVersion: 1.0.0
namespaceManifest: namespace.yaml
```

The name of the manifest file is arbitary here named "namespace.yaml".  It lives in the `template` folder and can be templated as previously mentioned.

KUDO is to add a `--create-namespace`.  The use of `--namespace` defines the namespace that will be used by the operator.  If the `--namespace` is missing and `--create-namespace` is provided it will result in a failure.  If a namespace manifest is provided and used in conjunction with `--create-namespace`, AND the namespace is missing from the cluster it will be created using the manifest file.  If the namespace exists, it is an error that will be reported back to the user.

The following rules apply:

1. No implicit creation of namespaces, namespace creation always require `--create-namespace`
1. If created namespace already exists in the cluster, it is a failure
1. If a namespace is being created and there is NO namespace manifest file, it will be created (if missing) as a simple namespace without metadata (perhaps we should consider adding "created by kudo details")
1. If a namespace is being created and there is a `namespaceManifest` file defined, the creation of the namespace will be with the metadata defined.
1. `name: {{ .Namespace }}` is not allowed in the manifest file and is considered a failure by the package verifier.  The namespace manifest is only useful to provide metadata to the namespace.  The control over what the name of the namespace  is controlled by KUDO only.
1. No support for static namespace or default namespace.
1. If no `--namespace` is provided the operator is installed to the "default" namespace.  The rules for `--create-namespace` are the same, if it exists, it will fail, otherwise it will be created.
1. If a namespace is created, KUDO waits until the namespace creation is completed to move forward with a plan.
1. If a namespace is needed to be created, it is detected early in the process prior to the deployment plan or as a first step.
1. All operator artifacts installed will go to this single namespace.
1. The namespace creation is done from the KUDO CLI and therefore the service account from the current user is used


### Proposal: Multi-Namespace (static)

There is NO support for multiple namespaces in a single operator or operators installed as dependencies.  Cluster-wide resource support is required to enable this feature.

### Proposal: Dependencies and Creating Namespaces

Regarding dependencies and creating namespaces, it is expected that the use of `--namespace` applies to the parent AND all dependent operators AND take the namespace is created prior to applying any dependent steps.  While there is likely desire to have independent control for dependent operators namespaces, this functionality will need to be thoughout more thoroughly regarding dependency management.  This part of the proposal was to provide completeness of namespace management inclusive of the current state of the [KEP-29 Operator Dependencies](0029-operator-dependencies.md) and not to extend further.  For the case where a parent and a dependent namespace manifest file is provided, only the parent namespace manifest file will be used.  All dependencies namespace manifests will be ignored.

### Proposal: Namespace Cleanup

There is NO consideration for namespace cleanup, even if KUDO created the namespace.

## Notes

1. It is important to note that cross namespace management of `ownerReferences` are not supported in Kubernetes.  [Owners and dependents documentation](https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/#owners-and-dependents) explicitly states that this is "disallowed by design".
1. It is unclear if we can delete a namespace, if we plan to manage this, then I would expect that we annotate or label the namespace as being managed by KUDO augmenting any manifest provided by the operator developer (which would need to be considered on a deepcopy).  At a future time, we can consider the deletion of namespaces by KUDO for namespaces it created and for which is absent of any operators.  That is not part of this current KEP.
1. Work Group Meeting on topic was recorded with details on the [community meeting notes](https://docs.google.com/document/d/19qveqaG5O4o1MouJmy2B23B2ehioAe8zGz3GgTJgRlI/edit#)

## Alternatives

### Proposal: No namespace creation

All namespace creation is managed outside KUDO. Users such as D2iQ would need to find or build automation for the creation of their namespaces. This REQUIRES that KUDO honor the deployment of resources to the "default" namespace, or to the namespace defined by `--namespace` during install.  

KUDO should further require that NO namespaces to be allowed in manifests for resources deployed during "install".  The concept of defining a manifest which lands in multiple namespaces is defined in a latter proposal and is a separate concern. Without multi-namespace support, all resources are installed in one namespace and namespace must be created prior to installation and that namespace must be specified during installation.
blank

This was rejected in favor of supporting the creation of namespaces.
