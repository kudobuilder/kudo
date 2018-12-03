# Comparison to other App Def tempaltes



##  Maestro

 Maestro focuses on mutli step (plan) applications that require the order to be specific to function



Developer Focused Vs App Operator Focused
Developer focused tools have great use case in sycning code with runtime for testing.  App Operators focus on Application pacakging for resubility/transparency/upgradability/application lifecycle management



## Extra compoenets deployed
Requires Extra components on cluster.  This was a big sticking point with "using Helm 2" from several sources.  From DC/OS side, it doesn't seem to be an issue sice the K8s cluster can have these components added by default without people having to do extra things.  


* Helm 2
* Maestro



## Parameters
The ability to take the same core definition and apply customization parameters to the definition and get a unique object.  Mosst tooling has been developed to address this issue.


## Reusable components/Patches to objects


Redefining the (almost) same components for several lifecycle phases can lead to bloated definition files and makes updates error prone.  

* Kustomize
* Maestro


## Custom Application lifecycles

Application Operators often want to perform maintancance on an application instance that is unique to the Framework.  The ability to define custom operatations on the framework as part of its definition makes common actions less error prone.  For example the mechanism for backing up an ElasticSearch index can be provided as part of the application Definition and used by Application Opertors without change.



## Custom Process Flows
Helm 3
Maestro - Either with steps/phases if possible, or by writing a program in ANY language that makes calls/commands to the K8s cluster and run that as a "job"


## CRDS
 The use of CRDs allow for many benefits:

### RBAC
What are the control mechanisms?  CRDs allow for RBAC control at a namespace level.  Expose Frameworks/Versions/Instances on a namepsace by namespace basis.  Allow creating of applications on a namespace basis.

### Finalizers


### Object Ownership (cascade delete/cleanup)

### Part of Namespace Backups



## Comparison Table




| Project | Defininition Language |Uses CRDs |  Dependencies | Multi Step |  Parameters |  Custom Lifecycles | Install Component|
|---------|----|-------|-----------|--|-------------------|----------------| --|
|Raw Yaml |YAML| No | No | No | No | No| No |
|Helm 3 | Lua | No | Yes | No | Yes | Yes | CLI |
|Helm 2|  Also lua? |No | ?? | No | Yes | No | CLI + Tiller |
| ksonnet|  jsonnet/libsonnet | No | ?? | No | Yes | ?? | CLI|
| OpenShift Templates| yaml| No | No | No | Yes | No | Just Openshift....|
| Kustomize | yaml | No | No | Yes | No | No| CLI|
|Maestro| Yaml + Kustomize | Yes | Yes | Yes |Yes | Yes | Yes

