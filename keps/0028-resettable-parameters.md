---
kep-number: 28
short-desc: A parameter flag to reset parameter values after a plan is executed
title: Resettable parameters
authors:
  - @aneumann82
owners:
  - @aneumann82
editor:
creation-date: 2020-03-31
last-updated: 2020-03-31
status: provisional
see-also:
replaces:
superseded-by:
---

## Summary

This KEP describes addition of a flag for parameters that allows a parameter to be reset after it was used in a plan.

## Motivation

The new flag allows an operator to define a parameter that is basically one-time-use. Especially for manually triggered
plans this will be an often used case:
- Start a backup with a specific name
- Evict a specific pod from a stateful set
- Start a repair plan for a specific node or datacenter

All these examples have in common that they require input parameters. The parameters are required, and the user should be
forced to set them. If we do not have resettable parameters, it might happen that a parameters are still set from a
previous execution and the user forgets to set it.

When parameters are marked as resettable, they are set to the default value after plan execution, and KUDO can error 
out if a required parameter is not set. 

### Goals

Make it possible to automatically reset a parameter after a plan is executed. 

### Non-Goals

- Set parameters to specific values (except for a default)
- Set parameters at other moments than at the end of a plan

## Proposal 1

Add an additional attribute `resettable` to parameter specifications in `params.yaml`:

```yaml
  - name: BACKUP_NAME
    description: "The name under which the backup is saved."
    resettable: "true"
```

The default value for this flag would be `false`.

If the flag is set to `true`, the parameter will be set to the default value when *any* plan finishes. This change
of parameter value should *not* trigger any plan execution. This is the preferred proposal.

## Proposal 2

An alternative would be a "string" type parameter that allows a user to set a specific plan after which the parameter
is reset:

```yaml
  - name: BACKUP_NAME
    description: "The name under which the backup is saved."
    resetAfterPlan: "backup"
```

This would reset the parameter after the plan `backup` is executed. The downside with this approach is that a parameter
could be set at some point and then be unknowingly used later.

### User Stories

- [#1395](https://github.com/kudobuilder/kudo/issues/1395) Resettable parameters

### Implementation Details/Notes/Constraints

- The parameter reset should happen if a plan reaches a terminal state, either `COMPLETED` or `FATAL_ERROR`.

### Risks and Mitigations



## Implementation History

- 2020-03-31 - Initial draft. (@aneumann)

## Alternatives

An alternative would be to have a new step type `SetParameter` that can modify a parameter and set it to a custom value.
This would allow a lot more flexibility, but also introduce a lot more complexity: Parameter values could then change
in the middle of a plan execution, triggering new plans might happen, etc. This might be an interesting idea for
an upcoming enhancement.
