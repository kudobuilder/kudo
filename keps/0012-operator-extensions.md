---
kep-number: 12
title: Operator Extensions
short-desc: How to extend existing operators. Adding or modifying tasks and plans, etc.
authors:
  - "@runyontr"
owners:
  - @runyontr
  - "@gerred"
editor: TBD
creation-date: 2019-06-18
last-updated: 2019-06-18
status: implementable
---

# Operator Extensions

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints-optional)
  - [Outstanding Questions](#outstanding-questions)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)

[tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Summary

Generic implementations of operators don't capture all customizations that are required for all end users. Rather than having to fork and maintain a patched version of an Operator, Extending an operator will allow for additions and customizations to be made to operators to meet additional business needs.

## Motivation

Extensions and bases describe a mechanism for building operators or extensions to operators from a given base. In this document, an **extension** is any KUDO operator that extends from some base. A **base** is the complete set of manifests, metadata, and other files provided by that base's type. A base should provide complete information that users of that base tool are expected to have. The base types and what they expose to operators that extend from them are described in their respective sub-sections.

### Goals

- Be able to extend an existing Operator

### Non-Goals

- Extending Helm Charts, CNAB or operators
- Managing repos of extensions

## Proposal

### User Stories

- As an Application Operator, I want to be able to add a custom Plan to an existing OperatorVersion.
- As an Application Operator, I want to be able to patch existing templates in an existing OperatorVersion.
- As an Application Operator, I want to be able to add or update parameters to an existing OperatorVersion.
- As an Application Operator, I want to be able to update my base operator easily.

### Extension Spec

To support extending from a base, `operator.yaml` is extended to support the `extends` keyword:

```yaml
extends:
  kudo:
    operator: "mysql"
    version: "5.7"
```

What gets inherited? Everything. An operator defined as:

```yaml
operator: bar
extends:
  kudo:
    operator: "foo"
    version: "1.0
```

will have the EXACT same functionality (plans, parameters, tasks, templates, etc) as the base `foo` operator.

#### Referencing an object from the base

#### Adding a Task

Tasks can be added via the same `tasks` spec in the `Operator` definition. Templates used to define the task can come from the base operator by pre-fixing `base/` to the template name, or from local templates as defined in [KEP-0009](0009-operator-toolkit.md). The follow example shows two new tasks called `load-data` and `clear-data` defined for the extension operator that uses both a template from the base, with a patch object that is defined in the extension operator.

```yaml
extends:
  kudo:
    operator: "mysql"
    version: "5.7"
version: "5.7"
tasks:
  - name: load-data
    kind: Apply
    spec:
      resources:
        - base/init.yaml
      patch:
        - load-data.yaml
  - name: clear-data
    kind: Delete
    spec:
      resources:
        - base/init.yaml
      patch:
        - load-data.yaml
```

#### Adding a Plan

Plans can be added to the extension operator and can reference tasks defined in either the base or the extension operator:

```yaml
plans:
  load:
    steps:
      - name: load
        tasks:
          - load-data
      - name: cleanup
        tasks:
          - clear-data
```

#### Modifying a Task

When a task is defined in an extended operator, it **replaces** the task from the base. The tasks available are dependent on the base type and are described more in detail in their corresponding sub-section.

In order to better extend and adjust tasks from the base, the `from` keyword will be supported. The `from` directive inside of a named task copies over all resources and patches from the base task of the same name. Resources and patches that overlap with base resource and patch names replace resource and patches already defined by the base.

Presuming there was a task in the base operator defined as:

```yaml
tasks:
  - name: init
    kind: Apply
    spec:
      resources:
        - init.yaml
```

An extension operator, that was trying to update the `init` task with a patch captured in `templates/init-patch.yaml` could update the task in different, but equivalent ways:

```yaml
tasks:
  - name: init
    kind: Apply
    spec:
      from: base/init
      patch:
        - init-patch.yaml
```

```yaml
tasks:
  - name: init
    kind: Apply
    spec:
      resources:
        - base/init.yaml
      patch:
        - init-patch.yaml
```

#### Modifying Plan

When a plan is defined in an extended operator, it **replaces** the plan from the base. All plans defined in the base are available in the extension plan.

#### Add and Updating Parameters

Parameters can be updated in the extension operator by providing new default values or descriptions. The follow example shows how parameters can be overridden and added from the base operator.

Consider the following parameter file defined in the base operator

```yaml
backup:
  default: backup.sql
  description: The file the backup job saves the sql dump, and the file the restore occurs from.
password:
  default: password
  description: Some words
```

And this file defined in the extension operator:

```yaml
backup:
  default: /path/to/new/location.sql
password:
  description: A more detailed description of the parameter
data:
  default: /path/to/sample/data.sql
  description: Storage location of sample data to load
```

Would combine in the extension operator as though the following parameters file was used:

```yaml
backup:
  description: The file the backup job saves the sql dump, and the file the restore occurs from.
  default: /path/to/new/location.sql
password:
  default: password
  description: A more detailed description of the parameter
data:
  default: /path/to/sample/data.sql
  description: Storage location of sample data to load
```

#### Example Operator Extension

This operator is built from the MySQL operator defined above, but adds custom plans that allow for the loading and clearing of data that is required for a particular business application, as well as a new parameter that allows for sizing the PVC that backups are stored on.

```bash
.
├── operator.yaml
├── params.yaml
└── templates
    ├── clear-data.yaml
    ├── pvc-size-patch.yaml
    └── load-data.yaml
```

In order to implement these changes, we need to add the plans for `load-data` and `clear-data` and update the jobs that backup and restore the data.

##### operator.yaml

`operator.yaml` is the base definition of an operator. It follows the following format, extracted from the MySQL example:

```yaml
extends:
  kudo:
    operator: "mysql"
    version: "5.7"
version: "5.7"
tasks:
  - name: backup
    kind: Apply
    spec:
      from: base/backup
      patch:
        - pvc-size-patch.yaml
  - name: restore
    kind: Apply
    spec:
      from: base/restore
      patch:
        - pvc-size-patch.yaml
  - name: load-data
    kind: Apply
    spec:
      resources:
        - base/init.yaml
      patch:
        - load-data.yaml
  - name: load-data-cleanup
    kind: Delete
    spec:
      resources:
        - base/init.yaml
      patch:
        - load-data.yaml
  - name: clear-data
    kind: Apply
    spec:
      from: base/init
      patch:
        - clear-data.yaml
  - name: clear-data-cleanup
    kind: Delete
    spec:
      from: base/init
      patch:
        - clear-data.yaml
plans:
  resize-pv:
    steps:
      - name: resize
        tasks:
          - pv
  load:
    steps:
      - name: load
        tasks:
          - load-data
      - name: cleanup
        tasks:
          - load-data-cleanup
  clear:
    steps:
      - name: clear
        tasks:
          - clear-data
      - name: cleanup
        tasks:
          - clear-data-cleanup
```

Tasks `load-data` and `clear-data` were created with the two different specifications for how to define a task. The `backup` and `restore` tasks were updated with the new patch this operator provides

##### params.yaml

This operator also provides a new parameter that can be used to specify unique datasources to load, and the size of the PVC that's used for backups.

```yaml
data-location:
  default: https://s3.aws.com/bucket/data.sql
  description: "Location of data to load into database"
  trigger: load
backup-pvc-size:
  default: 1Gi
  description: "Size of the PVC"
  trigger: resize-pv
```

### Implementation Details/Notes/Constraints

The implementation of extensions is independent of the ability to run non-KUDO defined
operators, however there are some relationships that need to be considered when extending an operator that has a different implementation engine. See forthcoming [KEP 0013](0013-external-specs.md)

### Outstanding Questions

- How are references done? Where do I look for the base operator? Installed in the cluster? Referenced by the CLI from somewhere? Contained in the dependency folder?
- Can the operator name change? Or does it have to stay the same? For example does a MySQL extension have to be of type MySQL to allow upgrades from existing MySQL Operators to it, or is it a completely different Operator?
- should initial implementation here have the `kudo` inside of the `extends` spec:

```yaml
extends:
  kudo:
    operator: "mysql"
    version: "5.7"
```

vs

```yaml
extends:
  operator: "mysql"
  version: "5.7"
```

## Graduation Criteria

Being able to implement the sample operator defined above.

## Alternatives

Instead of having an extension, we could require a forking and patching of any operator to allow for customization
