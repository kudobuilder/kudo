---
kep-number: 9
title: New KUDO package format
short-desc: Description of YAML operator definitions, features, plans/steps/tasks organization, templates and parameters
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
last-updated: 2019-06-27
status: implemented
see-also:
  - KEP-0002
---

# New KUDO package format

## Table of Contents

* [New KUDO package format](#new-kudo-package-format)
   * [Table of Contents](#table-of-contents)
   * [Summary](#summary)
   * [Motivation](#motivation)
   * [Goals](#goals)
   * [Proposal](#proposal)
      * [Definitions](#definitions)
      * [Operator Organization](#operator-organization)
         * [operator.yaml](#operatoryaml)
         * [params.yaml](#paramsyaml)
         * [common/](#common)
         * [templates/](#templates)
      * [Plans](#plans)
      * [Steps](#steps)
      * [Tasks](#tasks)
         * [Files](#files)
         * [Resources vs. Patches](#resources-vs-patches)
         * [Task Application](#task-application)
      * [Parameters](#parameters)
      * [Templates](#templates-1)
   * [Future Work](#future-work)
      * [Allow for other templating engines](#allow-for-other-templating-engines)

## Summary

Drive KUDO to have a set of tools and specifications for creating operators.

## Motivation

KUDO provides a way to reduce the amount of code necessary for the creation of an operator for Kubernetes. The current implementation of KUDO requires a significant amount of YAML to define an operator, located in one file. This YAML relies in nested YAML inlined via multiline strings, is not extensible, and is prone to error. This KEP defines an operator SDK that solves these issues.

## Goals

The toolkit must provide the ability for:

- conditional inclusions
- defining default values
- replacement of values
- removal of keys and values

## Proposal

A KUDO operator is split into several base components, ordered from the outside-in:

- [Plans](#plans)
- [Steps](#steps)
- [Tasks](#tasks)
- [Templates](#templates)
- [Parameters](#parameters)
- [Extensions and Bases](#extensions-and-bases)
- [Execution State](#execution-state)

These combine to form an operator responsible for the deployment, upgrading, and day 2 operations of software deployed on Kubernetes. While KUDO ships with a templating system, it is not intended to advance the state of deploying software on Kubernetes, and provides facilities for integrating with other templating and deployment systems. KUDO's focus is instead on the sequencing and day 2 operations of this software, and is able to take advantage of existing templating to fulfill this need.

### Definitions

### Operator Organization

An operator package is a folder that contains all of the manifests needed to create or extend a KUDO operator. In the most basic form, an operator bundle is structured in the following format:

```bash
.
├── operator.yaml
├── params.yaml
└── common
    └── common.yaml
└── templates
    ├── backup.yaml
    ├── clear-data.yaml
    ├── deployment.yaml
    ├── init.yaml
    ├── load-data.yaml
    ├── pvc.yaml
    ├── restore.yaml
    └── service.yaml
```

#### operator.yaml

`operator.yaml` is the base definition of an operator.  It follows the following reference format.

```yaml
name: "operator"
description: "operator desc"
version: "0.3.0"
appVersion: "v1.0-rc2"
kudoVersion: "0.8.0"
kubernetesVersion: "1.16"
maintainers:
  - name: "Billy Bob"
    email: "bb@kudo.dev"
url: "kudo.dev"
tasks:
  deploy:
    resources:
      - pvc.yaml
      - deployment.yaml
  validation:
    resources:
      - validation.yaml
plans:
  deploy:
    strategy: serial
    phases:
      - name: zookeeper
        strategy: parallel
        steps:
         - name: validation
           tasks:
             - validation
           delete: true
```

An example looks like:

```
name: "zookeeper"
version: 0.2.0
kudoVersion: 0.5.0
appVersion: 3.4.14
kubernetesVersion: 1.15.0
maintainers:
  - name: Tom Runyon
    email: runyontr@gmail.com
url: https://zookeeper.apache.org/
tasks:
  - name: deploy
    kind: Apply
    spec:
      resources:
        - pvc.yaml
        - deployment.yaml
        - service.yaml
        - deployment2.yaml
        - job.yaml
      patches:
        - deploy-patch.yaml
      patchesStrategicMerge:
        - super-weird-deploy-patch.yaml
  - name: init
    kind: Apply
    spec:
      resources:
        - init.yaml
  - name: cleanup
    kind: Delete
    spec:
      resources:
        - init.yaml
  - name: pv
    kind: Apply
    spec:
      resources:
        - pvc.yaml
  - name: backup
    kind: Apply
    spec:
      resources:
        - init.yaml
      patches:
        - backup.yaml
  - name: backup-cleanup
    kind: Delete
    spec:
      resources:
        - init.yaml
        - backup.yaml
  - name: restore
    kind: Apply
    spec:
      resources:
        - init.yaml
      patches:
        - restore.yaml
  - name: restore-cleanup
    kind: Delete
    spec:
      resources:
        - init.yaml
        - restore.yaml
  - name: load-data
    kind: Apply
    spec:
      resources:
        - init.yaml
      patches:
        - load-data.yaml
  - name: clear-data
    kind: Delete
    spec:
      resources:
        - init.yaml
      patches:
        - clear-data.yaml
  - name: query
    kind: Apply
    spec:
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
          - cleanup
  backup:
    steps:
      - name: pv
        tasks:
          - pv
      - name: backup
        tasks:
          - backup
      - name: cleanup
        tasks:
          - backup-cleanup
  restore:
    steps:
      - name: restore
        tasks:
          - restore
      - name: cleanup
        tasks:
          - restore-cleanup
```

While subsequent sections go into deeper detail, the top level keys of the operator are:

- **version**: String defining the version of a given operator
- **tasks**: A map of tasks that can be run. These are the atomic runnable unit of a KUDO operator, and are made up of a series of YAML manifests. These are defined more in detail in [Tasks](#tasks).
- **plans**: A map of plans that can be run. These are the core partitioning unit of a KUDO operator. A plan is intended to run a single "operations task" for an operator, such as backup, restore, deployment, or update. This is defined in detail in [Plans](#plans)

This file undergoes a Go template pass on Instance instantiation before being parsed. This is described more in detail in [Extensions and Bases](#extensions-and-bases)

#### params.yaml

The `params.yaml` file is a struct that defines parameters for operator. This can articulate descriptions, defaults, and triggers, etc. In the MySQL example, this looks like:

```yaml
apiVersion: kudo.dev/v1beta1
parameters:
  - name: "parameter_name"
    displayName: "parameter display"
    description: "parameter desc"
    required: true
    default: "param default"
    trigger: "backup"    
```

An example looks like:

```yaml
NODE_COUNT:
  description: "Number of nodes spun up for Zookeeper"
  default: 3
  displayName: "Node Count"

MEMORY:
  description: Amount of memory to provide to Zookeeper pods
  default: "1Gi"

CPUS:
  description: Amount of cpu to provide to Zookeeper pods
  default: "0.25"

STORAGE_CLASS:
  description: "The storage class to be used in volumeClaimTemplates. By default its not required and the default storage class is used."
  required: false
```

These values are meant to be overridden by Instance resources when instantiating an operator.

This file undergoes a Go template pass on Instance instantiation before being parsed. This is described more in detail in [Parameters](#parameters).

#### common/

The common directory contains YAML manifests for all instances of the operator to leverage. The requirement/scope of these objects is defined in [KEP 0005](0005-cluster-resources-for-crds.md)

#### templates/

The templates directory contains YAML manifests for operator tasks to use. These are described in more detail in [Templates](#templates-1).

Each template undergoes a Go template pass for an Instance when it's corresponding Task is run.

### Plans

Plans are the core unit of operation within a KUDO operator. Each KUDO plan represents an individual unit of operations work. This may include backups, restores, deployments, upgrades, compaction, or any other myriad of operations that an application operator may want to perform with KUDO.

A single plan is composed of [steps](#steps) and a step is composed of [tasks](#tasks). Individual steps are executed sequentially by default while the tasks within the step are executed in parallel.

### Steps

Steps are used for fine grained partition within a plan. Steps are represented as a list of objects that define the step name and a list of tasks to be run during that step.

All tasks within the same step are applied at the same time. For runnable templates within a task (such as a Deployment, StatefulSet, Pod, etc), parallelism of workloads should be controlled through other Kubernetes primitives such as PodDisruptionBudget, and added to the task where the relevant workload is run.

If a step contains a runnable task that has readiness probes defined, the step waits until the readiness probe succeeds before moving onto the next step.

### Tasks

Tasks are a map of task name to a list of templates that should be executed in that step. With [Extensions and Bases](#extensions-and-bases), tasks can be represented in multiple forms, as long as the end result is a ready-to-run YAML manifest for a Kubernetes object.

#### Files

If a filename is specified, KUDO will execute a Go Template on the relevant filename, described more in detail in [Templates](#templates-1).

#### Resources vs. Patches

KUDO additionally supports Kustomize for defining resources and patching for tasks. Kustomize is applied after any Go templating steps. This is useful for defining common bases for objects, or for extending operators as described in [Extensions and Bases](#extensions-and-bases).

#### Task Application

Once KUDO has assembled a full set of templates for a task, they will be applied using Kubernetes server-side apply at the same time.

### Parameters

Parameters are a map of Parameter structs, with the key being the name of the parameter. A default value can be set inside of a parameter, as well as a description. When no default is specified, the value is required. If an Instance is created without a required parameter, it will have an Error event added to it describing the validation failure. As described in [Execution State](#execution-state), they are wrapped into the `.Params` object for use by all templated objects. All keys, even arbitrary keys unused by KUDO, are wrapped into the `.Params` object for potential use by other templates.

Parameters are intentionally open for extension for adding fields in the future for validation and more.

### Templates

A template is a standard Kubernetes manifest which **MAY** have additional Go Templating. These Go Templates include [Sprig](https://github.com/masterminds/sprig) to provide a standard library of functions familiar to Helm users. Additional values are available and are described in [Execution State](#execution-state).

## Future Work

### Allow for other templating engines

It may the be case that an operator developer does not want to, or cannot leverage the current templating system, either because functionality is not present in the language, or the operator may need to query external systems for value injection. We may want to extend our supported templating system to include other rendering laguages (e.g. [cue](https://github.com/cuelang/cue)), or allow an operator to deploy their own rendering engine in the Kubernetes cluster and expose a well defined interface (e.g. defined with Swagger) to KUDO for send rendering requests.

The specifications of what this API needs to be is out of scope of this KEP.
