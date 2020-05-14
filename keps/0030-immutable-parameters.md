---
kep-number: 30
short-desc: Immutable parameters that can only be set on installation
title: Immutable parameters
authors:
  - @aneumann82
owners:
  - @aneumann82
editor:
creation-date: 2020-04-28
last-updated: 2020-05-14
status: provisional
see-also:
---

## Summary

This KEP describes a feature to allow parameters to be defined as immutable. Immutable parameters can not be updated after the installation of the operator

## Motivation

Certain aspects of an operator must be defined while installing an operator and can not be changed later on. 

For example parameters that end up defining PVCs or the podManagementPolicy of a stateful set can not be changed after they are applied. Changing these parameters would lead to a broken state of the operator where the resources can not be applied again until the parameters are set back to their original value. In these cases an immutable parameter would prevent a user from accidentally break the operator.

### Goals

Make it possible for an operator developer to specify that a parameter can not be changed after the installation

### Non-Goals

- Immutable Parameters that can be set after the installation - There is a use case for parameters that have an undefined state until they are first used, for example in a specific plan, but can not be changed afterwards. This KEP requires a parameter to receive a value at installation time, either by using a default or provided by the user.

## Proposal

An `immutable` property is added to the parameter definition. If that property is true, the parameter needs to either have a default value or has to be explicitly provided by the user when the operator is installed. The parameter can not be updated with `kudo update -p` anymore.

```yaml
parameters:
  - name: NUM_TOKENS
    description: The number of tokens on each node of the cluster
    default: 256
    immutable: true
```

The default value for `immutable` is `false`.

### User Stories

- [#652](https://github.com/kudobuilder/kudo/issues/652) Make it possible to specify immutable parameter

- NUM_TOKENS in the Cassandra operator
- STORAGE_CLASS in the Zookeeper operator
- DISK_SIZE, STORAGE_CLASS and PERSISTENT_STORAGE in the Kafka operator

### Implementation Details/Notes/Constraints

- KUDO CLI will not perform any checks with regards to immutability
- The admission webhook verifies if a parameter is immutable
  - If the value of any immutable parameter changes from the current value it rejects the whole update
  - If the value of all immutable parameters are the same as their current values it allows the update
- If no value for an immutable parameter is given on installation, KUDO copies the default value from the OperatorVersion into the instance. This makes sure that the parameter value will never change, even if instance is changed to use a newer OperatorVersion that has different defaults.
- If a parameter definition is immutable, it must either have a default value or needs to be marked as required.
- `k kudo package list parameters <operatorname>` will be extended to include the immutability (either by default or with an extra parameter):
```
Name         	Default	Required Immutable
CLIENT_PORT  	2181   	true     false
CPUS         	250m   	true     false
DISK_SIZE    	5Gi    	true     true    
NODE_COUNT   	3      	true     false
```

### Risks and Mitigations


## Alternatives

### `immutable` Section in `params.yaml`

Instead of adding an `immutable` property to the parameter definition, we could add an `immutable` section to the params.yaml file:

```yaml
parameters:
  - name: NODE_COUNT
    description: The number of nodes in the cluster
    default: 3
immutables:
  - name: NUM_TOKENS
    description: The number of tokens on each node of the cluster
    default: 256
```

- This would prevent grouping of parameters by functionality; some parameters that apply to a specific section of the operator may immutable
- It would make it harder to see if a parameter is immutable or not when the list of parameters grows larger than a screen. It will not be obvious that the editor is currently in the parameters or immutables section of the file

### A separate `immutables.yaml`

Instead of adding an `immutable` property to the parameter definition, we could add a separate file that contains all immutable parameters:

`params.yaml`:
```yaml
parameters:
  - name: NODE_COUNT
    description: The number of nodes in the cluster
    default: 3
```

`immutables.yaml`:
```yaml
parameters:
  - name: NUM_TOKENS
    description: The number of tokens on each node of the cluster
    default: 256
```

- The same issues as with the previous proposal apply

## Implementation History

- 2020-04-28 - Initial draft. (@aneumann)
- 2020-05-01 - Clarified point of check (webhook) and use of default values. (@aneumann)
- 2020-05-14 - Clarified and reworded some things, added alternatives (@aneumann)
