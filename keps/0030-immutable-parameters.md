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
last-updated: 2020-04-28
status: provisional
see-also:
replaces:
superseded-by:
---

## Summary

This KEP describes a feature to allow parameters to be defined as immutable. Immutable parameters can not be updated after the installation of the operator

## Motivation

Certain aspects of an operator must be defined while installing an operator and can not be changed later on. 

### Goals

Make it possible for an operator developer to specify that a parameter can not be changed after the installation

### Non-Goals

- Parameters that can be set once after the installation - this KEP requires a parameter to be set in the installation process.

## Proposal

An flag `immutable` is added to the parameter definition. If that flag is true, the parameter needs to be set (or have a default) when the operator is installed. The parameter can not be updated with `kudo update -p` anymore.

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

The KUDO CLI will check if a parameter is immutable when updating an instance and error out.

The admission webhook should verify if the parameter is immutable and reject the update to the instance on update.


Open Questions: 
- Should it be allowed to use an immutable parameter in a `kudo update` invocation when the value is the same as in the instance? This would effectively not be an update. 
  - Yes: The CLI would have to either check the current instance or rely on the admission webhook to error out
  - No: Would make using the same parameter file to install and update a KUDO instance impossible 


### Risks and Mitigations


## Implementation History

- 2020-04-28 - Initial draft. (@aneumann)

## Alternatives
