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

## Table of Contents

<!--ts-->
      * [Summary](#summary)
      * [Table of Contents](#table-of-contents)
      * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals)
      * [Proposal](#proposal)
         * [User Stories](#user-stories)
         * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
            * [Installation](#installation)
            * [Updates](#updates)
            * [Upgrades](#upgrades)
            * [Other](#other)
         * [Risks and Mitigations](#risks-and-mitigations)
      * [Alternatives](#alternatives)
         * [Immutable Section in params.yaml](#immutable-section-in-paramsyaml)
         * [A separate immutables.yaml](#a-separate-immutablesyaml)
      * [Implementation History](#implementation-history)

<!-- Added by: aneumann, at: Thu May 14 19:34:48 CEST 2020 -->

<!--te-->

## Motivation

Certain aspects of an operator must be defined while installing an operator and can not be changed later on. 

For example parameters that end up defining PVCs or the podManagementPolicy of a stateful set can not be changed after they are applied. Changing these parameters would lead to a broken state of the operator where the resources can not be applied again until the parameters are set back to their original value. In these cases an immutable parameter would prevent a user from accidentally break the operator.

### Goals

Make it possible for an operator developer to specify that a parameter can not be changed after the installation

### Non-Goals

- Constants that can be used in templating but can not be changed by the user. Immutable parameters can always be set by the user on installation.
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

#### Installation
- If a parameter definition is immutable, it must either have a default value or needs to be marked as required.
- If no value for an immutable parameter is given on installation, KUDO copies the default value from the OperatorVersion into the instance. This makes sure that the parameter value will never change, even if instance is changed to use a newer OperatorVersion that has different defaults.

#### Updates
- KUDO CLI will not perform any checks with regards to immutability on installation or updates
- The admission webhook verifies if a parameter is immutable
  - If the value of any immutable parameter changes from the current value it rejects the whole update
  - If the value of all immutable parameters are the same as their current values it allows the update, allowing for the use of the same parameter file for installation and update as long as the immutable parameters do not change.

#### Upgrades
- Upgrades to a new OperatorVersions:
  - A parameter can be made immutable in a newly released OperatorVersion.
    - When executing `kudo upgrade` the user has to explicitly set the value for parameters that are changed from mutable to immutable
    - If an parameter is changed from mutable to immutable and the user does not explicitly specify a parameter the upgrade will abort with an error message
    - This ensures that the user knows about the new immutability
  - A new immutable parameter can be added to the operator
    - When executing `kudo upgrade` the user has to explicitly set the value for the new parameter, even if the parameter has a default value
    - If the user does not provide a value on `kudo upgrade`, the upgrade will abort with an error message that contains the (optional) default value
    - This ensures that the user knows about the new parameter and that it can not be changed after the upgrade
  - A parameter can be made mutable in a newly released OperatorVersion
    - The parameter keeps the existing value
    - The value of the parameter can already be changed while running `kudo upgrade`
    - This change does not require explicit consent from the user
  - An immutable parameter can be removed
    - This change does not require explicit consent from the user
    - The value for a removed parameter will be removed from the installed instance

#### Other
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

### Immutable Section in `params.yaml`

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
- 2020-05-15 - Added section on upgrads (@aneumann)
