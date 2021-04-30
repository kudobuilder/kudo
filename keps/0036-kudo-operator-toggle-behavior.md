---
kep-number: 36
short-desc: KudoOperator Toggling
title: Parameter-controlled toggling of KudoOperator tasks
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

# Parameter-controlled toggling of KudoOperator tasks

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

<!-- [Tools for generating]: https://github.com/ekalinin/github-markdown-toc -->

## Summary

This KEP aims to add an ability to switch an operator dependency on and off
via a specified parameter of a parent operator. Namely, as a result of
implementing this KEP, `KudoOperator` task definition will get an optional field
containing a name of a boolean parameter (of the parent operator), the value
of which will determine whether the child operator instance should exist.

## Motivation
From [#1775](https://github.com/kudobuilder/kudo/issues/1775):
> Currently, KudoOperator tasks don't support toggle behavior which
> significantly complicates the handling of the optional dependencies. A good
> example could be Spark and History Server or Kafka and Schema Registry. The
> only possible options to handle it today are either include the templates
> as-is into the parent operator and control the contents at the template level
> or to implement a job (via Toggle task) which deploys/removes the dependency
> based on a parameter value. Also, cascading updates/upgrades become
> complicated in these scenarios.

### Goals

* Add an ability to enforce existence/**non-existence** of the child
  operator instance (and all its children) via a parent operator parameter,
  i.e. to toggle the dependency.

* Make parent operators specifying an invalid parameter for toggling fail
  instance admission (as opposed to simply failing a plan with misconfigured
  `KudoOperator` task).

### Non-Goals

* Chain validation of child instances. `KudoOperator` configurations that result
  in failed instance admission of a child do not fail parent instance admission,
  only plan execution. Functionality introduced by this proposal is also
  affected by this: for example, if a parent switches on a child that uses a
  non-existent parameter for toggling a grandchild, parent instance admission
  will succeed, only the corresponding parent plan will fail.

* OperatorVersion validation. This proposal adds one more way to create
  an operator that just cannot be installed regardless of instance parameters.

* Introducing an `UninstallKudoOperator` task. An ability to unconditionally
  remove a child operator (for example, installed by a previous version), while
  no more difficult, is not a goal of this proposal.

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
In this example, when the parameter `installDependency` equals to `true`, the
task will be handled as a normal, unconditional `KudoOperator`. When the
parameter equals to `false`, execution of the task will ensure that
the corresponding operator instance is not installed (uninstalling it if
necessary).

#### Validation
A `KudoOperator` task referencing a non-existing parent parameter will be
treated as invalid: instance admission (and plan execution, if instance
admission is not used) will fail.

Task execution and instance validation will require that the type
of the specified parameter is either "boolean" or a string convertible
to boolean according to rules used by Go's `strconv.ParseBool()`

#### Changing `enablingParameter` on upgrade
No special handling for upgrade is planned.

Some consequencees of this choice:
 * Dropping the parameter will convert the task into an unconditional
   `KudoOperator` managing the same instance.
 * Switching to a parent operator parameter with a different name will have
   an effect determined only by the value of the new parameter.
 * Dropping a `KudoOperator` task in one operator version and re-introducing
   a dependency with the same name with completely different parameters
   in some next version will still allow the new version to uninstall
   the formerly child operator by upgrade with setting the newly introduced
   enabling parameter to `false`. Operator developers will still need to
   consider the whole history of development of an operator to avoid unintended
   consequences of upgrades.

### Implementation Notes

Implementation is relatively straightforward, including the instance removal
case: the child instance to be removed will be identified via the same mechanism
as one used for identifying the instance to patch on upgrade/update.

## Risks and Mitigations

## Drawbacks

This proposal makes the operator developer API even less robust with regards
to developer errors than it is now (see [Non-Goals](#non-goals)).

## Alternatives

Currently existing alternative is conditionally creating a child operator
instance (and also Operator/OperatorVersion if needed) by means of `Toggle`
tasks. This is extremely cumbersome in the existing form, but might become much
more convenient if we are to implement a stable API for managing operators
via a single custom resource (aka CRD-based installation; KEP yet to be filed).

However, one might argue that `KudoOperator` will still provide a much more
convenient framework for cases when a parent operator has multiple children
that are never used on their own.
