# Controller Responsibilities

There are several controllers deployed as part of KUDO.  This guide attempts to outline what each controller is responsible for


## Framework

### Permissions

* Framework
  - ["list", "get"]
* FrameworkVersion
  - ["list", "get"]
* CRDs
  - all

### Watches

* CRDs
* FrameworkVersions
* Frameworks

### Actions

#### New Framework

* Validation of Framework Spec

* If a `FrameworkVersion` for the `Framework` is present, it creates the FrameworkCRD.
** Attaches label = `heritage=kudo`
** Owner = Framework object
** Create ClusterRoleBinding for using Framework CRD

#### Delete Framework

Deletes the Framework CRD (if present) for the deleted `Framework`

#### New FrameworkVersion

If this is the first `FrameworkVersion` for the linked `Framework`, the Controller creates the Framework CRD

* Attaches label = `heritage=kudo`
* Owner = Framework object
* Create ClusterRoleBinding for using Framework CRD

If this is not the first `FrameworkVersion` for the `Framework`, the controller updates the Framework CRD to include the new `Version` as a member of the Framework CRD `Versions` field.

#### Delete FrameworkVersion

The Framework Controller removes the FrameworkVersion from the `Versions` field in the Framework CRD.

If no `Versions` remain, the controller deletes the Framework CRD


## FrameworkVersion

### Permissions

- Framework
  - ["list", "get"]

### Watches

* Frameworks
* FrameworkVersions


### Actions

#### New FrameworkVersion

* Validation of the `FrameworkVersion` spec
* Ensure referenced `Framework` exists
* Set OwnerRef of `FrameworkVersion` as Framework (????)

#### New Framework

* Identify any `FrameworkVersions` that reference new `Framework`, and set as `OwnerRef`

## KUDO - Replaces `Instance` Controller

The Instance controller will watch all CRDs that were genreated by Kudo.

### Permissions

- PlanExecution
  ["list","get","watch","update","create"]
- instance
  ["list","get","watch"]
- CRDs
  ["list"]
- FrameworkCRDs
  ["list","get"]

### Watches

* PlanExecution
* Instance
* CRDs

### Actions

#### NewInstance

Create a new PlanExecution for the Instance running the `deploy` plan specified in the FrameworkVersion.

#### DeleteInstance

Call a cleanup plan (TODO).
Then cascade delete the objects

#### PlanExecution Status Update

As the current Plan updates, update the status of the Instance to reflect the current status of the plan.

#### New CRD

If CRD contains the `heritage=kudo` label, then watch new CRD

#### Delete CRD

If CRD contains the `heritage=kudo` label, then unwatch CRD

## PlanExecution

### Permissions
- Like everything that is Namespaced (no cluster scoped objects)
Create/update/delete/list/watch/get on all objects that could be referenced as part of a FrameworkVersion.  Includes coreapi, as well as
Instances and other non-kudo CRDs.  

### Watches

- Like everything.  

### Actions

#### Update on any object

Need to identify if this object was deployed as part of a Kudo plan ( owner of object == Instance).  If so, then call Reconcile on the Instance and/or PlanExecution.

#### New PlanExecution

Run the plan exeuction (expand on this section and/or reference Plan spec)


#### Suspend PlanExecution

* Update the plan as Suspended, and stop trying to continue the operations and/or check the health of the objects.
  