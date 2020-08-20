---
kep-number: 28
short-desc: Transient parameters that are not saved permanently in the instance
title: Transient parameters
authors:
  - @aneumann82
owners:
  - @aneumann82
editor:
creation-date: 2020-03-31
last-updated: 2020-04-28
status: provisional
see-also:
replaces:
superseded-by:
---

## Summary

This KEP describes addition of transient parameters that allows a parameter to be used only in a certain plan without saving the value in the instance.

## Motivation

Transient parameters allow a specific use-case that is basically one-time-use. Especially for manually triggered plans this will be an often used case:
- Start a backup with a specific name
- Evict a specific pod from a stateful set
- Start a repair plan for a specific node or datacenter

All these examples have in common that they require input parameters. The parameters are required, and the user should be forced to set them. If we do not have transient parameters, it might happen that a parameters are still set from a previous invocation and the user forgets to set it. Additionally, using persistent parameters for these use cases would clutter the instance parameter store with values that are not required to be saved.

When parameters are transient, they should be set to their default value after plan execution, and KUDO can error out if a required parameter is not set. 

### Goals

Make it possible to have transient parameters that are only available withing a single plan execution. 

### Non-Goals

- Set parameters to specific values (except for a default)
- Set or change parameters at other moments than at the end of a plan

## Proposals

#### Persistent vs. Transient parameters

Add an attribute `transient` to parameter specifications in `params.yaml`:

```yaml
  - name: BACKUP_NAME
    description: "The name under which the backup is saved."
    transient: "true"
```

The default value for this flag would be `false`, all parameters that are defined without the attribute are persistent by default.

If the flag is set to `true`, the parameter is transient and its value will not be persisted in the instance resource.

Transient parameters can be easily identified and will only be stored in the `planExecution` section of the instance. As the `planExecution` is reset after the plan finishes, the transient parameters from the plan execution will be discarded as well.

- Transient parameters cannot be `immutable`
  - They are reset after every plan execution and are therefore mutable by design
- Transient parameters can have a default value
- Transient parameters cannot be `required`
  - This would be a desired property and could be theoretically allowed. But transient parameters are often plan specific, and the `required` scope is global, it would be mostly too wide of a scope. Having plan-specific required parameter might be the scope of a different KEP.

Pros:
- Easy extension for parameters from the definition point of view
- Parameters are not bound to specific plans. Multiple plans can use the same transient parameters. (For example `backup` and `restore` could use the same `BACKUP_NAME` parameter)

Cons:
- The parameters for a specific plan are not separated from persistent parameters - It can be easy to miss the attribute.
- Without plan specific parameters, it is hard to determine if a plan requires a specific parameter. It will be easy for a user to forget to specify a transient parameter, and `kudoctl` will not be able to determine that a parameter is required to render a plan-specific resource.

#### Plan specific parameters

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

The parameters would only be valid during the execution of that plan and can only be used with the defining plan. As with the previous proposal, they can be stored in the `planExecution` part of the instance where the get discarded after plan execution.

To specify the parameter, it would have to be prefixed with the plan name in which it is defined:
`kudo plan trigger <instance> backup -p backup.NAME=MyBackup`

Plan parameters will be stored in a separate map named `PlanParams` which contains a map for the currently executed plan.
```
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ $.PlanParams.backup.NAME }}
```

Pros:
- Would work well for manually triggered plans
- The parameter scope would be clearly defined
- Transient and persistent parameters can be used at the same time 
- A prefix will make it clear which type is used when invoking `kudo install`, `kudo update` or `kudo plan trigger` 

Cons:
- Could potentially increase the size of the operator.yaml (If parameters are defined there)
- If the same parameter is used for two different plans, it would have to be repeated. ( for example BACKUP_NAME, used for backup and restore plans)
- The format of `{{ $.PlanParams.backup.NAME }}` is problematic if the same resource is used in multiple plans
- Plan specific parameters are not necessarily transient - Some persistent parameters may be plan specific as well
- More complex implementation (New PlanParams map, enhanced verification to make sure that resources which use a plan parameter are only used in the correct plans, ...)

Plan specific parameters may not really be the solution for the User stories in this KEP, but they might be useful in other circumstances.

### Transient parameters only on `kudo plan trigger`

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
  - If yes, it may allow setting persistent parameters with `kudo plan trigger`. The question would be if the definition would be more like Proposal 1 or 3 

Pros:
- Potentially no change to operator definition

### Implementation Details/Notes/Constraints

- The parameter value will be discarded when plan reaches a terminal state, either `COMPLETED` or `FATAL_ERROR`.
- Transient parameters will be saved in the `planExecution` structure in the instance CRD.


### Risks and Mitigations


## User Stories

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

It could be an option to set persistent parameters as well on plan trigger:

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
- The command line includes transient and persistent parameters
- The transient parameters should not be saved in the instance

## Implementation History

- 2020-03-31 - Initial draft. (@aneumann)
- 2020-04-03 - Added alternatives and user stories
- 2020-04-17 - Added proposal 5
- 2020-04-28 - Moved two proposals to alternatives, renamed KEP
- 2020-08-05 - Improved wording, cleaned up naming

## Alternatives

### Parameter reset after plan finish

This is a variant of the previous proposal: Plans will have a list of parameters that are reset when a plan is finished:

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
- It will be possible for a parameter to be transient for one plan but persistent for another. It would not be possible to store transient parameters in the `planExecution` structure of the instance

### SetParameters Task

Add new task type, `SetParameters`:
```yaml
  - name: backup-done
    kind: SetParameters
    spec:
      params:
        - name: 'RESTORE_FLAG'
           value: nil
```
This would be a very powerful task, but it would also provide a lot more ways to introduce complexity: Parameter values could change inside a plan execution, the change could trigger additional plans, etc.

Pros:
- Very powerful
- Would allow easy extension to set parameters to custom values

Cons:
- Very complex
- Parameters could change while the plan is executed
- What happens when a plan is triggered by a changed parameter