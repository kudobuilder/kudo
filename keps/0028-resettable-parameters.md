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

The new flag allows an operator to define a parameter that is basically one-time-use. Especially for manually triggered plans this will be an often used case:
- Start a backup with a specific name
- Evict a specific pod from a stateful set
- Start a repair plan for a specific node or datacenter

All these examples have in common that they require input parameters. The parameters are required, and the user should be forced to set them. If we do not have resettable parameters, it might happen that a parameters are still set from a previous execution and the user forgets to set it.

When parameters are marked as resettable, they are set to the default value after plan execution, and KUDO can error out if a required parameter is not set. 

### Goals

Make it possible to automatically reset a parameter after a plan is executed. 

### Non-Goals

- Set parameters to specific values (except for a default)
- Set parameters at other moments than at the end of a plan

## Proposal 1 - Reset flag on parameter

Add an additional attribute `resettable` to parameter specifications in `params.yaml`:

```yaml
  - name: BACKUP_NAME
    description: "The name under which the backup is saved."
    resettable: "true"
```

The default value for this flag would be `false`.

If the flag is set to `true`, the parameter will be set to the default value when *any* plan finishes. This change of parameter value should *not* trigger any plan execution. This is the preferred proposal.

An alternative would be a "string" type parameter that allows a user to set plans after which the parameter is reset:

```yaml
  - name: BACKUP_NAME
    description: "The name for backups to create or restore."
    resetAfterPlan: [ "backup", "restore" ]
```

This would reset the parameter after the plan `backup` is executed. The downside with this approach is that a parameter could be set at some point and then be unknowingly used later.

Pros:
- Both variants would be an easy extension for parameters from the definition point of view

Cons:
- The parameters for a specific plan are not separated from "normal" parameters
- It's not easy to determine 

## Proposal 2 - SetParameters Task

Add new task type, `SetParameters`:
```yaml
  - name: backup-done
    kind: SetParameters
    spec:
      params:
        - name: 'RESTORE_FLAG'
           value: nil
```
This is a lot more powerful, but also provides a lot more ways to introduce complexity: Parameter values could change inside a plan execution, what about triggered plans from param changes, etc.

Pros:
- Very powerful
- Would allow easy extension to set parameters to custom values

Cons:
- Very complex
- Parameters could change while the plan is executed
- What happens when a plan is triggered by a changed parameter

## Proposal 3 - Plan specific parameters

Specific plan parameters: These parameters would only be valid inside a specific plan and could be defined inside the operator:

```yaml
plans:
  backup:
    params:
      - name: 'NAME'
        description: "The name under which the backup is saved."
    strategy: serial
    phases:
      - name: nodes
        strategy: parallel
        steps:
          - name: node
            tasks:
              - backup
```

Alternatively it might be possible to define this in the `params.yaml`
```yaml
  - name: backup.NAME
    plan: backup
    description: "The name under which the backup is saved."
```

These parameters would only be valid during the execution of that plan.

To use the parameter, it would have to be prefixed with the plan name in which it is defined:
`kudo update <instance> -p backup.NAME=MyBackup`

```
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ $.PlanParams.backup.NAME }}
```

Pros:
- Would work well for manually triggered plans
- The parameter scope would be clearly defined

Cons:
- Could potentially increase the size of the operator.yaml
- If the same parameter is used for two different plans, it would have to be repeated. ( for example BACKUP_NAME, used for backup and restore plans)

## Proposal 4 - Parameter reset after plan finish

Define a list of parameters to be reset when a plan is finished:

```yaml
plans:
  backup:
    resetAfterPlan:
      - BACKUP_NAME
    strategy: serial
    phases:
      - name: nodes
        strategy: parallel
        steps:
          - name: node
            tasks:
              - backup
```

Pros:
- Does not add a new flag on parameter definition

Cons:
- It won't be obvious from the parameter list that this is a plan specific parameter

## Proposal 5 - Transient parameters only on `kudo plan trigger`

This proposal would not require new configuration options in the operator definition. Normal parameter definitions can be used, the handling depends on the usage:

```bash
kudo update --instance op-instance -p PARAM1=value
```
This would set a parameter persistently, the parameter value will be saved in the Instance CRD

```bash
kudo plan trigger someplan --instance op-instance -p PARAM1=value
```
This would use a parameter transiently, the value would be available while rendering templates, but the value would not be saved in the instance CRD. When the plan execution ends, the parameter value is discarded.

Open Questions:
- Should transient parameters marked in the parameter definition? 
  - If not, they could be set permanently with `kudo update`, which would reverse most of the ideas of transient parameters
  - If yes, it may allow setting permanent parameters with `kudo plan trigger`. The question would be if the definition would be more like Proposal 1 or 3 


### User Stories

- [#1395](https://github.com/kudobuilder/kudo/issues/1395) Resettable parameters

#### A backup plan for the Cassandra Operator

A plan that starts a backup for the whole cluster which is manually triggered

It has a `BACKUP_NAME` parameter that specifies the name of a backup that is to be created.  
This parameter needs to be unique on every execution and should not be reused. If the parameter is not unset after the backup plan is finished, a user could forget to set it again for the next execution.

The expected initiation is:

`kudo plan trigger --instance cassandra backup -p BACKUP_NAME=NewBackup`

- A call without defined parameter should fail.
- The parameter BACKUP_NAME should not be stored in the instance and must be set the next time the backup plan is triggered.

The operator has another parameter, `BACKUP_PREFIX`. This parameter describes the prefix that is used in the S3 bucket. This parameter should be saved in the instance, as it usually does not change often.

It could be an option to set permanent parameters as well on plan trigger:

`kudo plan trigger --instance cassandra backup -p BACKUP_NAME=2020-06 -p BACKUP_PREFIX=monthly`

- BACKUP_PREFIX would be saved in the instance, BACKUP_NAME would be transient

This is not a requirement, it's also possible to make two calls:
```bash
kudo update --instance cassandra -p BACKUP_PREFIX=monthly
kudo plan trigger --instance cassandra -p BACKUP_NAME=2020-06
```

#### The restore operation in the Cassandra Operator

The restore operation on the Cassandra Operator can be used to create a new cluster from an existing backup.

It uses a RESTORE_FLAG parameter that can be used on installation of a new C* cluster to restore an existing backup, and additional parameters like RESTORE_OLD_NAME and RESTORE_OLD_NAMESPACE to define which backup should be restored. 
When RESTORE_FLAG is set, the deployment includes an initContainer that downloads the data and prepares the new cluster to use this data.
The initContainer is only used on the very first start of the cluster and should not exist on subsequent restarts of the nodes, additionally the RESTORE_FLAG, RESTORE_OLD_NAME and RESTORE_OLD_NAMESPACE are useless/not used after the initial deploy plan is done.

The usage would be:

`kudo install cassandra -p RESTORE_FLAG=true -p RESTORE_OLD_NAME=OldCluster -p RESTORE_OLD_NAMESPACE=OldNamespace -p NODECOUNT=3 ...`

- Plan is `deploy`, used on installation
- The command line includes transient and permanent parameters
- The transient parameters should not be saved in the instance

### Implementation Details/Notes/Constraints

- The parameter reset should happen if a plan reaches a terminal state, either `COMPLETED` or `FATAL_ERROR`.

### Risks and Mitigations



## Implementation History

- 2020-03-31 - Initial draft. (@aneumann)
- 2020-04-03 - Added alternatives and user stories
- 2020-04-17 - Added proposal 5

## Alternatives

