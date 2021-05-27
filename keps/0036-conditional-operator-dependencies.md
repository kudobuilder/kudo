---
kep-number: 36
short-desc: Allowing existence of a child operator to be conditional on a parameter value
title: Conditional Operator Dependencies
authors:
  - "@asekretenko"
owners:
  - "@asekretenko"
editor: @asekretenko
creation-date: 2021-04-30
last-updated: 2021-05-27
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

This KEP aims to make it possible to create operators with optionally existing child operator instances. Namely, as a result of implementing this KEP, it will be possible to create a task, execution of which, depending on a value of a specified parameter, will ensure either
 * that a child operator instance exists, with appropriate parameters and operator version
 * or that the child instance does not exist (by cleaning it up if it does)

## Motivation

`KudoOperator` task type greatly simplifies handling operator dependencies by introducing the notion of a child operator. However, `KudoOperator` cannot be used directly when the existence of a child operator would have to be optional. Good examples of such cases are running Spark with an optional History Server or Kafka with an optional Schema Registry.

In both cases, it would be more convenient to control the optional features via separate child operators; however, as of today, there are only two ways to have such a component installed optionally: either to include templates as-is into the parent operator and to control the contents at the template level, or to implement a job (via a `Toggle` task) which deploys/removes the optional component based on a parameter value. Cascading updates/upgrades become complicated in these scenarios. (See [#1775](https://github.com/kudobuilder/kudo/issues/1775)).

### Goals

* Add an ability to toggle a `KudoOperator` task between ensuring existence (as it does now) and ensuring **non-existence** of the child operator instance (and its children) via a parent operator parameter. Toggling in both directions will be possibe.

### Non-Goals

* Adding a way to turn a child instance into an independent one (i.e. an ability to "switch off" an existing dependency relationship between two instances).

* Adding a way to turn an independent instance into a child instance.

* Preventing users from modifying parent operator parameters that are only used to configure the child operator when the child instance does not exist. In the Spark History Server example, history server options will be configured via a parameter of a parent operator; switching off the history server operator will have no effect on the user's ability to change the value of the "history server options" parameter of the parent operator.

* Displaying whether the child operator is enabled or disabled in an output of `kudo plan status`. As for now, the latter only descends to the status of individual plan steps of the parent instance and tells nothing about the child instances; this KEP does not aim to change this.

* Chain validation of child instances. As for now, `KudoOperator` task executions that result in failed instance admission of a child do not fail parent instance admission, only plan execution. Functionality introduced by this proposal is also affected by this: for example, if a parent switches on a child that uses a non-existent parameter for toggling a grandchild, parent instance admission will succeed, only the corresponding parent plan will fail.

* OperatorVersion validation. This proposal adds one more way to create an operator version that cannot be installed regardless of instance parameter values.

* Introducing a task that unconditionally removes a child operator instance.

* Conditional optionality. If, for example, an operator A has two optional children B and C, but C is not needed if B is not enabled, the user will need to disable C separately, which does not provide a good user experience. However, in this specific example operator developer could resolve the problem by turning C into a child of B or, if they have no control over the code of B, by adding a layer of indirection: a conditionally installed operator B\* that will unconditionally install B and conditionally install C. Conversely, if enabling B always requires C to be enabled, then B could be turned into a child of C.

## Proposal

### Interface
An optional field `enablingParameter` will be added to the `KudoOperator` task:
```
- name: child
  kind: KudoOperator
  spec:
    package: "./packages/child"
    parameterFile: child-params.yaml
    enablingParameter: "installChild"
```
In this example, when the parameter `installChild` equals to `true`, the task will be handled as a normal, unconditional `KudoOperator`. When the parameter equals to `false`, execution of the task will ensure that the corresponding operator instance is not installed (uninstalling it if necessary).

#### Validity criteria
A `KudoOperator` task specifying a non-existing parent parameter via `enablingParameter` will be treated as invalid: operator package validation, instance admission and task execution will all fail in this case.

Instance admission and task execution will require that the type of the specified parameter is either "boolean" or a string convertible to boolean according to rules used by Go's `strconv.ParseBool()`.

### Implementation Notes

Implementation is relatively straightforward, including the instance removal case: the child instance to be removed will be identified via the same mechanism as one used for identifying the instance to patch on upgrade/update.

## Risks and Mitigations

## Drawbacks

This proposal makes the operator developer API even less robust with regards to developer errors than it is now (see [Non-Goals](#non-goals)).

## Alternatives

Currently existing alternatives:
 * Pulling the would-be contents of a child operator into the main operator as `Apply` tasks with heavily parametrized templates, or as `Toggle` tasks. This becomes complicated when the would-be child operator has to implement something more complex than a simple unconditional set of `Apply`/`Toggle` tasks.
 * Conditionally creating/deleting child operator instance by means of `Toggle` tasks. This is extremely cumbersome in the existing form, but might become much more convenient if we are to implement a stable API for managing operators via a single custom resource (aka CRD-based installation; KEP yet to be filed). However, one might argue that `KudoOperator` is still going to provide a much more convenient framework for cases when a parent operator has multiple children that are never used on their own.
