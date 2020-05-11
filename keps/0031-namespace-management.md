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
last-updated: 2020-05-11
status: draft
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
* [Proposals](#proposals)
    * [Proposal 1: No namespace creation](#proposal-1-no-namespace-creation)
    * [Proposal 2: Namespace Creation](#proposal-2-namespace-creation)
      * [Alternative Ideas](#alternative-ideas)
    * [Proposal: Multi-Namespace (static)](#proposal-multi-namespace-static)
    * [Proposal: Dependencies and Creating Namespaces](#proposal-dependencies-and-creating-namespaces)
* [Notes](#notes)
* [Alternatives](#alternatives)


## Summary

This KEP aims to provide a definition on namespace management and provide guidance and constraints a KUDO operator may use namespaces.  This includes the expectations of creation, deletion, defining how an operator will install to a namespace and the ability of an operator to leverage more than one namespace.

## Motivation

Like many Kubernetes Object, there is an expectation that a KUDO operator is organized by namespace.  This KEP defines expectations around namespace creation and use.

### Goals

* Define if a namespace is a prerequistie to operator installation
* Define if a namespace is created by KUDO
* Define support for advance namespaces (namespaces with metadata)
* Define if an operator can be installed into a new namespace
* Define if an operator can leverage more than one namespace and under what conditions


### Non-Goals


## Namespace Management in the Eco-System

### Kubernetes Objects and Namespaces

When working kubernetes objects, be it Pods, ReplicaSets, Deployments and the like that require a namespace, the responsbility falls on the user to create that namespace prior assigning the resource or it will fail.  Available is an [auto-provision admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#namespaceautoprovision) available to auto-create namespaces. Users desiring auto-provisioning of namespaces should install this tool.  This webhook has the limitation of creating namespaces without extra metadata which defeats the value for use cases which need that metadata.

### Helm

Helm v1 and v2 provided an auto-provision capability.  The new Helm v3 (based on previous experiences) removed this capability and now requires users to create namespaces prior to installing a helm chart. Much detail is written in comments for [helm issue 6794](https://github.com/helm/helm/issues/6794)

### The Many Meanings of a Namespace

Challenages to auto-provision of namespaces are complicated by the many reasons / values for creating them which includes:

1. Logical separation of work (separating 1000s of pods into 10s of buckets for management)
1. Quota management
1. Security separation
1. Metadata (where labels and annotations provide meaning and value to controllers managing resources in a namespace)

Based on the last 3 reasons, it is common for an administrator to create namespaces to manage the other concerns associated with a namespace.  It would be challeging to manage a cluster where dozen or more namespaces are adhoc being created (or deleted) which may require a host of other configurations around quota and security.

## Proposals

Prior to making KEP `provisional` we will chose a namespace creation proposal to move forward with and move the 2nd proposal to alternative.  For now, we have 2 competing namespace creation proposals.  
In addition there is a proposal for the use of multiple namespaces.

### Proposal 1: No namespace creation

All namespace creation is managed outside KUDO. Users such as D2iQ would need to find or build automation for the creation of their namespaces. This REQUIRES that KUDO honor the deployment of resources to the "default" namespace, or to the namespace defined by `--namespace` during install.  

KUDO should further require that NO namespaces be allow in manifests for resources deployed during "install".  The concept of defining manifest which land in multiple namespaces is defined in a latter proposal and is a separate concern. Without multi-namespace support, all resources are install to one namespace and namespace must be created prior to installation and that namespace must be specified during installation.

### Proposal 2: Namespace Creation

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

The template supports templates for labels and annotations, however the `name:` metadata is NOT allowed to be set or controlled through templating.   This is to prevent the "templating" of the namespace name such as `name: {.Namespace}-extended`.  KUDO will add the `name:` for metadata when there is a need to create a namespace with metadata using the namespaceManifest file. The `operator.yaml` file is to will be extended to support `namespaceManifest`.  

```
# operator.yaml
name: "kubeflow"
operatorVersion: "0.2.0"
kudoVersion: "0.10.1"
kubernetesVersion: 1.15.0
appVersion: 1.0.0
namespaceManifest: templates/namespace.yaml
```

KUDO is to add a `--apply-ns`.  The use of `--namespace` defines the namespace that will be used by the operator.  If the namespace is missing than a failure will occur.  If used in conjunction with `--apply-ns`, than the namespace if missing from the cluster will be created using the manifest file.  If the namespace exists, the metadata will be applied to the existing namespace overwriting the exiting metadata.

The follow rules apply:

1. Missing `namespaceManifest` means that there is no metadata.  Using `--apply-ns` will create namespace without metadata.
1. `name: {{ .Namespace }}` is not allowed in the manifest file and is considered a failure by the package verifier.  The namespace manifest is only useful to provide metadata to the namespace.  The control over what the name of the namespace is is controlled by KUDO.
1. No support for static namespace.
1. If no `namespaceManifest` and no `--namespace` the operator installs to "default".  If used in conjunction with `--apply-ns`, it will be ignored.  There is no overwritting of the "default" namespace.
1. If `--namespace` provided and namespace doesn't exist, if `--apply-ns` is provided KUDO creates it.
1. If KUDO creates namespace, if there is a `namespaceManifest` it uses it to create namespace otherwise does a simple namespace creation.
1. If KUDO has `namespaceManifest` and `--apply-ns` is used there is no checks, the manifest file is applied to the namespace (exception is "default").
1. if a namespace is created KUDO waits until the namespace is created to move forward in a plan.
1. if a namespace is needed to be created, it is detected early in the process prior to the deployment plan or as a first step.

#### Alternative Ideas

1. Do we allow for a default namespace other than "default"
1. We could have another flag to enforce creation, meaning `--namespace` by itself will use that namespace but does NOT create and in combination with `--force` or `--ns-ensure` will create namespace.
1. namespace comparison could be simple check.  It is unclear when an overwrite should occur or if this should be an error.
1. Auto create (with a `--apply-ns` or `--ns-ensure`)
1. Creation of namespace only handled if `namespaceManifest` is provided.  The case of simple namespace (without metadata) would need to be considered.

### Proposal: Multi-Namespace (static)

There is a desire to support multiple namespaces. The desire seems to stem from 2 sources; KUDO early adoptors as operator devs what have large multi-operator needs and those using KUDO for micro-service management. An example of large operator development is Kubeflow, where there is 20+ operators inside D2iQ KUDO Kubeflow operator.  There is a strong desire to logically group many of the supportive infrastructure operators inside their own namespace, reducing the cognitive burden of the end user by removing non-primary operators into a separate namespace from the primary operators.
 In this model, the expectations are that the 2nd tier namespaces (not the "primary" operator) will have statically defined namespaces which should be honored.   A significant challenge to this is objects with No namespace will land whereever KUDO is configured to install the operator, but other objects will have no control and will be static.   Part of the justification of this is kubeflow, where there are 30+ operators, many which are supportive in nature that the operator developer would like to section off.

This propose is for KUDO to deploy objects to KUDO managed {{.Namespace}} if not specified and to honor `namespace:` for objects that have them.  There are challenges to package validation as we won't know what is accidently missing or accidently specified.  There are challenges to it being static which can be overcome through the use of Params.  It seems if Params are used, that the values for the instance are  immutable after installation.


### Proposal: Dependencies and Creating Namespaces

Regarding dependencies and creating namespaces, it is expected that the use of `--namespace` applies to the parent AND all dependent operators AND take the namespace is created prior to applying any dependent steps.  While there is likely desire to have independent control for dependent operators namespaces, this functionality will need to be thoughout more thoroughly regarding dependency management.  This part of the proposal was to provide completeness of namespace management inclusive of the current state of the [KEP-29 Operator Dependencies](0029-operator-dependencies.md) and not to extend further.

## Notes

1. It is important to note that cross namespace management of `ownerReferences` are not supported in Kubernetes.  [Owners and dependents documentation](https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/#owners-and-dependents) explicitly states that this is "disallowed by design".
1. It is unclear if we can delete a namespace, if we plan to manage this, then I would expect that we annotate or label the namespace as being managed by KUDO augmenting any manifest provided by the operator developer (which would need to be considered on a deepcopy)

## Alternatives

Intentionally left blank
