---
title: Comparison
menu: docs
---
# Comparison to other App Def templates


##  KUDO

 KUDO focuses on multi-step (plan) applications that require the order to be specific to function



Developer Focused Vs App Operator Focused
Developer focused tools have great use case in sync-ing code with runtime for testing.  App Operators focus on Application packaging for reusability/transparency/upgradability/application lifecycle management


## Extra components deployed

Requires Extra components on cluster.  This was a big sticking point with "using Helm 2" from several sources.  From DC/OS side, it doesn't seem to be an issue since the K8s cluster can have these components added by default without people having to do extra things.  

* Helm 2
* KUDO


## Parameters

The ability to take the same core definition and apply customization parameters to the definition and get a unique object.  Most tooling has been developed to address this issue.


## Reusable components/Patches to objects

Redefining the (almost) same components for several lifecycle phases can lead to bloated definition files and makes updates error prone.  

* Kustomize
* KUDO


## Custom Application lifecycles

Application Operators often want to perform maintenance on an application instance that is unique to the Operator.  The ability to define custom operations on the operator as part of its definition makes common actions less error prone.  For example the mechanism for backing up an ElasticSearch index can be provided as part of the application Definition and used by application Operators without change.


## Custom Process Flows

Helm 3
KUDO - Either with steps/phases if possible, or by writing a program in ANY language that makes calls/commands to the K8s cluster and run that as a "job"


## CRDS

The use of CRDs allow for many benefits:

### RBAC

What are the control mechanisms?  CRDs allow for RBAC control at a namespace level.  Expose Operators/Versions/Instances on a namepsace by namespace basis.  Allow creating of applications on a namespace basis.

### Finalizers

### Object Ownership (cascade delete/cleanup)

### Part of Namespace Backups


## Discoverable Repo for Applications

It should be easy to convert a Helm chart into a OperatorVersion since we can just "render" the chart.  Additionally we plan to build the Universe Shim to accept any DC/OS operator.  Thus we should be able to pull from either of these public repos of apps (and any internally hosted app site)


## Comparison Table

|                 Project | Definition Language | Uses CRDs | Dependencies | Multi Step | Parameters | Custom Lifecycles | Install Component | App Repo |
| ----------------------: | :-----------------: | --------: | -----------: | ---------: | :--------: | ----------------: | :---------------: | -------: |
|            **Raw Yaml** |        YAML         |        No |           No |         No |     No     |                No |        No         |       No |
|              **Helm 3** |         Lua         |        No |          Yes |         No |    Yes     |               Yes |        CLI        |      Yes |
|              **Helm 2** |      Also Lua?      |        No |           ?? |         No |    Yes     |                No |   CLI + Tiller    |      Yes |
|             **ksonnet** |  jsonnet/libsonnet  |        No |           ?? |         No |    Yes     |                ?? |        CLI        |       No |
| **OpenShift Templates** |        yaml         |        No |           No |         No |    Yes     |                No |     Openshift     |       No |
|           **Kustomize** |        YAML         |        No |           No |        Yes |     No     |                No |        CLI        |       No |
|                **KUDO** |  YAML + Kustomize   |       Yes |          Yes |        Yes |    Yes     |               Yes |        Yes        |      Yes |
