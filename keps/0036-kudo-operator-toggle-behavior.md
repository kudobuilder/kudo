---
kep-number: 36
short-desc: Allowing existence of an operator dependency to be conditional on a parameter value
title: Conditional Operator Dependencies
authors:
  - "@asekretenko"
owners:
  - "@asekretenko"
editor: @asekretenko
creation-date: 2021-04-30
last-updated: 2021-04-30
status: provisional
see-also:
  - KEP-23
  - KEP-29
---

# Conditional Operator Dependencies

## Table of Contents
* [Summary](#summary)
* [Motivation](#motivation)
  * [Goals](#goals)
  * [Non-Goals](#non-goals)
* [Proposal](#proposal)
  * [Interface](#interface)
     * [Validation](#validation)
     * [Changing enablingParameter on upgrade](#changing-enablingparameter-on-upgrade)
  * [Implementation Notes](#implementation-notes)
  * [Risks and Mitigations](#risks-and-mitigations)
* [Drawbacks](#drawbacks)
* [Alternatives](#alternatives)


## Summary

This KEP aims to make it possible to create operators with conditionally existing child operators. Namely, as a result of implementing this KEP, it will be possible to create a task, execution of which, depending on a value of a specified parameter, will ensure either
 * that a child operator instance exists, with appropriate parameters and operator version (this is what `KudoOperator` task currently does)
 * or that the child instance does not exist (by cleaning it up if it does)

## Motivation

While `KudoOperator` task type greatly simplifies handling operator dependencies by introducing the notion of a child operator, it is almost of no help when one needs to make existence of a child operator conditional. Good examples of cases where this is necessary are running Spark with an optional History Server or Kafka with an optional Schema Registry.

In both cases, it would be more convenient to control the optional dependencies via separate child operators; however, the only ways to optionally have a dependency today are either to include templates as-is into the parent operator and to control the contents at the template level, or to implement a job (via a `Toggle` task) which deploys/removes the dependency based on a parameter value. Also, cascading updates/upgrades become complicated in these scenarios. (See [#1775](https://github.com/kudobuilder/kudo/issues/1775)).

### Goals

* Add an ability to toggle a `KudoOperator` task between ensuring existence (as it does now) and ensuring **non-existence** of the child operator instance (and its children) via a parent operator parameter. Toggling in both directions will be possibe.

### Non-Goals

* Chain validation of child instances. `KudoOperator` configurations that result in failed instance admission of a child do not fail parent instance admission, only plan execution. Functionality introduced by this proposal is also affected by this: for example, if a parent switches on a child that uses a non-existent parameter for toggling a grandchild, parent instance admission will succeed, only the corresponding parent plan will fail.

* OperatorVersion validation. This proposal adds one more way to create an operator that just cannot be installed regardless of instance parameters.

* Introducing an `UninstallKudoOperator` task. An ability to unconditionally remove a child operator (for example, installed by a previous version), while no more difficult, is not a goal of this proposal.

## Proposal

### Interface
An optional field `enablingParameter` will be added to the `KudoOperator` task:
```
- name: dependency
  kind: KudoOperator
  spec:
    package: "./packages/dependency"
    parameterFile: dependency-params.yaml
    enablingParameter: "installDependency"
```
In this example, when the parameter `installDependency` equals to `true`, the task will be handled as a normal, unconditional `KudoOperator`. When the parameter equals to `false`, execution of the task will ensure that the corresponding operator instance is not installed (uninstalling it if necessary).

#### Validation
A `KudoOperator` task referencing a non-existing parent parameter will be treated as invalid: instance admission (and plan execution, if instance admission is not used) will fail.

Task execution and instance validation will require that the type of the specified parameter is either "boolean" or a string convertible to boolean according to rules used by Go's `strconv.ParseBool()`.

#### Changing `enablingParameter` on upgrade
No special handling for upgrade is planned.

Some consequencees of this choice:
 * Dropping the parameter will convert the task into an unconditional `KudoOperator` task managing the same instance.

 * Switching to a parent operator parameter with a different name will have an effect determined only by the value of the new parameter.

### Implementation Notes

Implementation is relatively straightforward, including the instance removal case: the child instance to be removed will be identified via the same mechanism as one used for identifying the instance to patch on upgrade/update.

## Risks and Mitigations

## Drawbacks

This proposal makes the operator developer API even less robust with regards to developer errors than it is now (see [Non-Goals](#non-goals)).

## Alternatives

Currently existing alternatives:
 * Pulling the would-be contents of a child operator into the main operator as `Apply` tasks with heavily parametrized templates, or as `Toggle` tasks. This becomes complicated when the would-be child operator has to implement something more complex than a simple unconditional set of `Apply`/`Toggle` tasks.
 * Conditionally creating/deleting child operator instance by means of `Toggle` tasks. This is extremely cumbersome in the existing form, but might become much more convenient if we are to implement a stable API for managing operators via a single custom resource (aka CRD-based installation; KEP yet to be filed). However, one might argue that `KudoOperator` is still going to provide a much more convenient framework for cases when a parent operator has multiple children that are never used on their own.
