---
kep-number: 9
title: KUDO Operator Toolkit
authors:
  - "@kensipe"
  - "@gerred" 
  - "@zen-dog"
  - "@alenkacz"
  - "@runyontr"
owners:
  - "@gerred"
editor: "@gerred"
creation-date: 2019-05-14
last-updated: 2019-05-23
status: provisional
see-also:
  - KEP-0002
---

# KUDO Operator Toolkit

## Table of Contents

- [KUDO Operator Toolkit](#kudo-operator-toolkit)
  - [Table of Contents](#table-of-contents)
  - [Summary](#summary)
  - [Motivation](#motivation)
  - [Goals](#goals)
  - [Proposal](#proposal)
    - [Definitions](#definitions)
    - [Operator Organization](#operator-organization)
      - [operator.yaml](#operatoryaml)
      - [params.yaml](#paramsyaml)
      - [templates/](#templates)
    - [Plans](#plans)
    - [Steps](#steps)
    - [Tasks](#tasks)
      - [Files](#files)
      - [Webhooks](#webhooks)
      - [Resources vs. Patches](#resources-vs-patches)
      - [Task Application](#task-application)
    - [Parameters](#parameters)
    - [Templates](#templates)
  - [Extensions and Bases](#extensions-and-bases)
    - [Helm](#helm)
  - [Execution State](#execution-state)
  - [Example Framework](#example-framework)

## Summary

Drive KUDO to have a set of tools, frameworks and specifications for creating operators.

## Motivation

KUDO provides a way to reduce the amount of code necessary for the creation of an operator for Kubernetes. The current implementation of KUDO requires a significant amount of YAML to define an operator, located in one file. This YAML relies in nested YAML inlined via multiline strings, is not extensible, and is prone to error. This KEP defines an operator SDK that solves these issues.

## Goals

The toolkit must provide the ability for:

* conditional inclusions
* defining default values
* replacement of values
* removal of keys and values

## Proposal

A KUDO framework is split into several base components, ordered from the outside-in:

* [Plans](#plans)
* [Steps](#steps)
* [Tasks](#tasks)
* [Templates](#templates)
* [Parameters](#parameters)
* [Extensions and Bases](#extensions-and-bases)
* [Execution State](#execution-state)

These combine to form an operator responsible for the deployment, upgrading, and day 2 operations of software deployed on Kubernetes. While KUDO ships with a templating system, it is not intended to advance the state of deploying software on Kubernetes, and provides facilities for integrating with other templating and deployment systems. KUDO's focus is instead on the sequencing and day 2 operations of this software, and is able to take advantage of existing templating to fulfill this need.

### Definitions


### Operator Organization

An operator bundle is a folder that contains all of the manifests needed to create or extend a KUDO operator. In the most basic form, an operator bundle is structured in the following format:

```
.
├── operator.yaml
├── params.yaml
└── templates
    ├── backup.yaml
    ├── deployment.yaml
    ├── init.yaml
    ├── pvc.yaml
    ├── restore.yaml
    └── service.yaml
```

#### operator.yaml

`operator.yaml` is the base definition of an operator. It follows the following format, extracted from the MySQL example:

```yaml
version: "5.7"
tasks:
  deploy:
    resources:
    - deployment.yaml
    - pvc.yaml
    - service.yaml
  init:
    resources:
      - init.yaml
  pv:
    resources:
      - backup-pv.yaml
  backup:
    resources:
      - base-job.yaml
    patches:
      - backup.yaml
  restore:
    resources:
      - restore.yaml
  load-data:
    resources:
      - job.yaml
  clear-data:
    resources:
      - job.yaml
  query:
    resources:
      - job.yaml
plans:
  deploy:
    steps:
      - name: deploy
        tasks:
        - deploy
      - name: init
        tasks:
        - init
      - name: cleanup
        tasks:
        - init
        delete: true
  backup:
    triggers:
      - parameter: backupFile
    steps:
      - name: pv
        tasks:
        - pv
      - name: backup
        tasks:
        - backup
      - name: cleanup
        tasks:
        - backup
        delete: true
  restore:
    steps:
      - name: restore
        tasks:
        - restore
      - name: cleanup
        tasks:
        - restore
        delete: true
```

While subsequent sections go into deeper detail, the top level keys of the operator are:

* **version**: String defining the version of a given operator
* **tasks**: A map of tasks that can be run. These are the atomic runnable unit of a KUDO operator, and are made up of a series of YAML manifests. These are defined more in detail in [Tasks](#tasks).
* **plans**: A map of plans that can be run. These are the core partitioning unit of a KUDO operator. A plan is intended to run a single "operations task" for an operator, such as backup, restore, deployment, or update. This is defined in detail in [Plans](#plans)

This file undergoes a Go template pass on Instance instantiation before being parsed. This is described more in detail in [Extensions and Bases](#extensions-and-bases)

#### params.yaml

The `params.yaml` file is a list of keys and values for operator parameters and their defaults. In the MySQL example, this looks like:

```yaml
backupFile: backup.sql
password: password
```

These values are meant to be overridden by Instance resources when instantiating an operator.

This file undergoes a Go template pass on Instance instantiation before being parsed. This is described more in detail in [Parameters](#parameters).

#### templates/

The templates directory contains 0-to-many YAML manifests for operator tasks to use. These are described in more detail in [Templates](#templates).

Each template undergoes a Go template pass for an Instance when it's corresponding Task is run.

### Plans

Plans are the core unit of operation within a KUDO operator. Each KUDO plan represents an individual unit of operations work. This may include backups, restores, deployments, upgrades, compaction, or any other myriad of operations that an application operator may want to perform with KUDO.

A single plan is composed of [Steps](#steps).

### Steps

Steps are used for fine grained parition within a plan. Steps are represented as a list of objects that define the step name and a list of tasks to be run during that step.

All tasks within the same step are applied at the same time. For runnable templates within a task (such as a Deployment, StatefulSet, Pod, etc), parallelism of workloads should be controlled through other Kubernetes primitives such as PodDisruptionBudget, and added to the task where the relevant workload is run.

If a step contains a runnable task that has readiness probes defined, the step waits until the readiness probe succeeds before moving onto the next step.

### Tasks

Tasks are a map of task name to a list of templates that should be executed in that step. With [Extensions and Bases](#extensions-and-bases), tasks can be represented in multiple forms, as long as the end result is a ready-to-run YAML manifest for a Kubernetes object.

#### Files
If a filename is specified, KUDO will execute a Go Template on the relevant filename, described more in detail in [Templates](#templates).

#### Webhooks
If a webhook is specified, KUDO will POST the listed webhook with a JSON body containing the full KUDO [execution state](#execution-state). The webhook **MUST** respond with the following:

* Status code: 200
* Content-type: application/json

The body of this response **MUST** be a fully-qualified Kubernetes object in JSON form. If multiple objects are required, they **MUST** be wrapped in a Kubernetes List API object. The contents of a list object **CAN** be composed of multiple Kubernetes objects of different types.

Webhooks **CAN** have side effects to control external state or to fetch external parameters for use in their templates, as long as the response is still a Kubernetes object.

Webhook HTTP connections time out after 30 seconds, after which a Timeout event is added to the PlanExecution the task is running on.


#### Resources vs. Patches

KUDO additionally supports Kustomize for defining resources and patching for tasks. Kustomize is applied after any Go templating steps. This is useful for defining common bases for objects, or for extending frameworks as described in [Extensions and Bases](#extensions-and-bases).

#### Task Application

Once KUDO has assembled a full set of templates for a task, they will be applied using Kubernetes server-side apply at the same time.

### Parameters

Parameters are a key-value list of parameter names and their defaults. As described in [Execution State](#execution-state), they are wrapped into the `.Params` object for use all templated objects.

### Templates

A template is a standard Kubernetes manifest while, which **MAY** have additional Go Templating. These Go Templates include [Sprig](https://github.com/masterminds/sprig) to provide a standard library of functions familiar to Helm users. Additional values are available and are described in [Execution State](#execution-state).

## Extensions and Bases

### Helm

## Execution State

## Example Framework