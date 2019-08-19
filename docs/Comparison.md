---
title: Comparison
menu: docs
---

# KUDO

KUDO, a [Template Defined Application](#template-defined-application) tool developed with a focus on operators rather than developers, is used for multi-step (plan) applications that require the order of operations to be specific in order to function.

## Comparison To Other Kubernetes Application Lifecycle Management Tools

### Application Operator Focused

Other tools in this space are developer focused with the expectation that a developer wants to deploy and/or manage the lifecycle of their application or applications.
Application Operators (AppOps) focus more packaging applications for reusability, transparency, upgradability, and application lifecycle management beyond the simple CRUD operations.

KUDO's focus on AppOps fills a need that has not been well represented.

### In Cluster Components

KUDO requires a component installed in the cluster, like Helm v2 requires Tiller.
This has been a controversial issue througout the community due to usability and security concerns.
Because of these concerns, Helm v3, ksonnett, kustomize, etc, operate outside the cluster and thus only have the permissions of the person executing the command.
KUDO and Helm v2 operate as cluster administrators.

### Parameters

Compared with each of the other tools, KUDO is designed to have more robust parameterization with defaults, descriptions, and validation as part of the parameter spec.

### Custom Application Functions

AppOps often want to perform maintenance on an application that is unique to that application.
KUDO's ability to define custom operations as part of its definition makes common actions less error prone.
For example, backing up an ElasticSearch index can be provided as part of the application Definition and used by AppOps without change.

### Custom Process Flows

KUDO allows custom process flows with steps/phases if possible, or by writing a program in ANY language that makes calls/commands to the K8s cluster and run that as a "job".
Helm (v2 and v3) is the only other tool in this space that allows custom flows.

### Kubernetes Custom Resource Definitions

The use of [CRD]s allow for many benefits:

#### Role Based Access Control

By using CRDs instead of opening up a gRPC service like it's Tiller counterpart, the KUDO operator takes advantage of Kubernetes native support of [RBAC] to provide permissions isolated to a namespace.

#### Finalizers

With KUDO's dependency management, [Finalizers][finalizers] can be placed on KUDO Operators that are depended on by other applications.
Zookeeper, for example, may have a Finalizer placed on it by the Kafka KUDO operator to prevent Zookeeper from being removed whike Kafka is still using it.
This functionality is unique to KUDO.

#### Object Ownership (cascade delete/cleanup)

By marking ownership of resources installed by KUDO to the custom resource that created them, cascading deletes allow for more complete cleanup.
If the custom resource is deleted, all the resources that were created because of that are also deleted.
This functionality is unique to KUDO.

#### Part of Namespace Backups

Because of the use of CRDs, the definition of the state of your application is part of the state of the cluster.
Backing up a namespace (using a tool like Velero for example) will also backup the defitions associated with that namespaces desired state, allowing the KUDO operator to reconcile that state upon restoration.

### Discoverable Repo for Applications

It should be easy to convert a Helm chart into an OperatorVersion since we can just "render" the chart.
Additionally we plan to build the Universe Shim to accept any DC/OS operator.
Thus we should be able to pull from either of these public repos of apps (and any internally hosted app site).

## Comparison Table

|         Project | Definition Language | Uses CRDs | Dependencies | Multi Step | Parameters | Custom Lifecycles | Install Component | App Repo |
| ----------------------: | :-----------------: | --------: | -----------: | ---------: | :--------: | ----------------: | :---------------: | -------: |
|      **Raw Yaml** |    YAML     |    No |      No |     No |   No   |        No |    No     |    No |
|       **Helm 3** |     Lua     |    No |     Yes |     No |  Yes   |        Yes |    CLI    |   Yes |
|       **Helm 2** |   Also Lua?   |    No |      ?? |     No |  Yes   |        No |  CLI + Tiller  |   Yes |
|       **ksonnet** | jsonnet/libsonnet |    No |      ?? |     No |  Yes   |        ?? |    CLI    |    No |
| **OpenShift Templates** |    yaml     |    No |      No |     No |  Yes   |        No |   Openshift   |    No |
|      **Kustomize** |    YAML     |    No |      No |    Yes |   No   |        No |    CLI    |    No |
|        **KUDO** | YAML + Kustomize  |    Yes |     Yes |    Yes |  Yes   |        Yes |    Yes    |   Yes |

## Reference

### Template Defined Application

Template defined application tools define the state of an application through a template.
Consider [Helm Charts][charts], [Kustomization][kustomization], and the [Cloud Native Application Bundle Specification][cnab].

[CRD]:https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/
[RBAC]:https://kubernetes.io/docs/reference/access-authn-authz/rbac/
[finalizers]:https://book.kubebuilder.io/reference/using-finalizers.html
[charts]:https://github.com/helm/charts
[kustomization]:https://github.com/kubernetes-sigs/kustomize/blob/master/docs/workflows.md
[cnab]:https://github.com/deislabs/cnab-spec