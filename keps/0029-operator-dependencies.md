---
kep-number: 29
title: Operator Dependencies
short-desc: Introducing operators depending on other operators
authors:
  - "@zen-dog"
  - "@porridge"
owners:
  - "@zen-dog"
editor: @zen-dog
creation-date: 2020-03-30
last-updated: 2020-04-16
status: implemented
---

# Operator Dependencies

## Table of Contents

* [Summary](#summary)
* [Motivation](#motivation)
  * [Goals](#goals)
  * [Non-Goals](#non-goals)
* [Proposal](#proposal)
  * [Implementation Details](#implementation-details)
    * [KudoOperator Task](#kudooperator-task)
    * [Deployment](#deployment)
      * [Client-Side](#client-side)
      * [Server-Side](#server-side)
      * [Parameterization](#dependencies-parametrization)
    * [Update](#update)
    * [Upgrade](#upgrade)
    * [Uninstalling](#uninstalling)
    * [Other Plans](#other-plans)
  * [Risks and Mitigation](#risks-and-mitigation)
* [Alternatives](#alternatives)

## Summary

This KEP aims to improve operator user and developer experience by introducing operator dependencies.

## Motivation

Recent operator development has shown that complex operators often depend on other operators to function. One workaround commonly seen can be found in the [Flink Demo](https://github.com/kudobuilder/operators/tree/main/repository/flink/docs/demo/financial-fraud) where an `Instance` of Flink needs an `Instance` of Kafka, which in turn needs an `Instance` of Zookeeper. There are two parts to the workaround:
1. the operator user needs to first **manually** install the `kafka` and `zookeeper` operators while skipping the instance creation (using the `kubectl kudo install ... --skip-instance` CLI option).
2. then, the user runs a regular `kubectl kudo install flink`, whose `deploy` plan includes `Instance` resources for Kafka and Zookeeper, which are bundled along other Flink operator templates in YAML format (rather than being created on-the-fly from an operator package).

This KEP is aiming at streamlining this experience for users and developers.

### Goals

Dependencies can be a complex topic. This KEP is not trying to boil the dependency ocean but rather limits itself to installation dependencies only, i.e. a set of `Operators/Instances` being installed together and removed together as a unit.

### Non-Goals

Dependency on an already running `Instance` is a non-goal. It is easy to imagine a situation when a new operator (e.g Kafka) may want to depend on the existing Zookeeper instance. However, such life-cycle dependency presents major challenges e.g. what happens when Zookeeper is removed? What happens when Zookeeper is upgraded, and the new version is incompatible with the current Kafka `Instance`? How can we ensure the compatibility? This KEP deliberately ignores this area and instead focuses on installation dependencies. Additionally, this KEP does not address output variables, referencing `Instance` resources, or installing other dependencies operators other than KUDOs own.

## Proposal

KUDO operators **already have** a mechanism to deal with installation dependencies called [plans, phases, and steps](https://kudo.dev/docs/developing-operators/plans.html#overview) with serial or parallel execution strategy. This mechanism is already powerful enough to express any dependency hierarchy including transitive dependencies (see more about this in [implementation details](#implementation-details)). The core of this proposal is to reuse this mechanism and extend it with the ability to install operators.

### Implementation details

#### KudoOperator Task

A new task kind `KudoOperator` is introduced which extends `operator.yaml` with the ability to install dependencies. Let's take a look at the Kafka+Zookeeper example:

```yaml
apiVersion: kudo.dev/v1beta1
name: "kafka"
operatorVersion: "1.3.1"
kudoVersion: 0.12.0
kubernetesVersion: 1.14.8
appVersion: 2.5.0
url: https://kafka.apache.org/
tasks:
  - name: zookeeper-operator
    kind: Operator
    spec:
        package: zookeeper

  ...

plans:
  deploy:
    strategy: serial
    phases:
      - name: deploy-zookeeper
        strategy: serial
        steps:
          - name: zookeeper
            tasks:
              - zookeeper-operator
      - name: deploy-kafka
        strategy: serial
        steps:
          ...
```

The `zookeeper-operator` task specification is equivalent to `kudo install zookeeper` CLI command which installs the Zookeeper package from the official repo. Here is a complete `Operator` task specification:

```yaml
tasks:
- name: demo
  kind: KudoOperator
  spec:
    package: # required, either repo package name, local package folder or an URL to package tarball
    appVersion: # optional, a specific app version in the official repo, defaults to the most recent one
    operatorVersion: # optional, a specific operator version in the official repo, defaults to the most recent one
    instanceName: # optional, the instance name
```

As you can see, this closely mimics the `kudo install` CLI command [options](https://github.com/kudobuilder/kudo/blob/main/pkg/kudoctl/cmd/install.go#L56) because at the end the latter will be executed to install the operator. We omit `parameters` and `parameterFile` options at this point as they are discussed in detail [below](#dependencies-parametrization).

#### Deployment

##### Client-Side

Upon execution of `kudo install ...` command with the above operator definition, CLI will:

1. Collect all operator dependencies by analyzing the `deploy` plan of the top-level operator and compose a list of all operator dependencies (tasks with the kind `Operator`) including **transitive dependencies**
2. Install all collected packages **skipping the instances** (same as `kudo install ... --skip-instance`). This step creates `Operator` and `OperatorVersion` resources. Note that since we do not create instances here we can install them in any order
3. Proceed with the installation of the top-level operator as usual (create `Operator`, `OperatorVersion` and `Instance` resources)

Since we do this step on the client-side we have access to the full functionality of the `install` command including installing operators from the file system. This will come very handy during the development and debugging which arguably becomes more complex with dependencies.

##### Server-Side

Upon receiving a new operator Instance with dependencies KUDO mangers workflow engine will:

1. Build a [dependency graph](https://en.wikipedia.org/wiki/Dependency_graph) by transitively expanding top-level `deploy` plan using operator-tasks as vertices, and their execution order (`a` needs `b` to be installed first) as edges
2. Perform cycle detection and fail if circular dependencies found. We could additionally run this check on the client-side as part of the `kudo package verify` command to improve the UX
3. If we haven't found any cycles, start executing the top-level `deploy` plan. When encountering an operator-task, apply the corresponding `Instance` resource. Here it is the same as for any other resource that we create: we check if it is healthy and if not, end current plan execution and "wait" for it to become healthy. KUDO manager already has a [health check](https://github.com/kudobuilder/kudo/blob/main/pkg/engine/health/health.go#L78) for `Instance` resources implemented.

Let's take a look at an example. Here is a simplified operator `AA` with a few dependencies:

```text
AA
├── BB
│   ├── EE
│   │   ├── H
│   │   └── I
│   ├── F
│   └── GG
│       ├── J
│       └── K
├── CC
│   ├── L
│   └── M
└── D

Legend:
- Operators and operator-tasks are marked with double letters e.g. 'AA' or `BB`
- other tasks are marked with single letters e.g. 'D'
- direct children of an operator are the 'deploy' plan steps e.g. for 'AA' deploy steps are 'BB', 'CC' and 'D'
```

In the first step we build a dependency graph. A set of all graph vertices (which are task-operators) `S` is defined as `S = {AA, BB, CC, EE, GG}`. A transitive relationship `R` between the vertices is defined as `(a, b) ∈ S` meaning _`a` needs `b` deployed first_. The transitive relationship for the above example is: `R = { (AA,BB), (AA,CC), (BB,EE), (BB,GG) }`. The resulting topological order `O` is therefor `O = (EE, GG, BB, CC, AA)` which has no cycles.

The instance controller (IC) then starts with the execution of the top-level `deploy` plan of the operator `AA`. The first task is the `BB` operator-task. When executing it, IC creates the `Instance-BB` resource and ends current reconciliation. Next, IC notices new `Instance-BB` resource, starts new reconciliation, and executes the  `deploy` plan of the operator `BB` which then creates `Instance-EE` resource. This way we are basically performing the depth-first search for the dependency graph, executing each vertex in the right order e.g. `EE` has to be healthy before `BB` deploy plan can continue with the next step `F`.

We would additionally add the higher-level `Instance` reference (e.g. `AA`) to the `ownerReferences` list of its direct children `Instance`s (e.g. `BB` and `CC`). This would help with determining which `Instance` belongs to which operator and additionally help us with [operator uninstalling](#uninstalling).

The status of the execution can be seen as usual as part of the `Instance.Status`. We could additionally forward the status of a dependency `Instance` to the top-level `Instance.Status` to simplify the overview.

Note that in the above example if e.g. `EE` and `CC` task-operators reference the same operator package KUDO will create two distinct instances (`Instance-EE` and `Instance-CC`). Otherwise, one of them would be a life-cycle dependency which is a [non-goal](#non-goals) for this KEP. There is also a naming issue here: an operator developer would have to give both instances distinct `spec.instanceName`s in order for them not to collide. Another issue is: what happens when a user installs the top-level operator in the same namespace twice? We should make sure dependencies instance names do not collide with each other. A suggestion would be to make instance names hierarchical so that e.g. `Instance-EE` would be named `aa.bb.ee` where `aa` is the name of the top-level instance of the operator `AA`. This will also improve the UX as users would be able to immediately know where the instance belongs to.

##### Dependencies Parametrization

We want to encourage operator composition by providing a way of operator encapsulation. In other words, operator users should not be allowed to arbitrarily modify the parameters of embedded operator instances. The higher-level operator should define all parameters that its **direct** dependency  operators need. Let's demonstrate this on an example of a simple operator `AA` that has operator `BB` as a dependency. 

```yaml
AA
└── BB
```
Operator `BB` has a required and empty parameter `PASSWORD`. To provide a way for the `AA` operator user to set the password we extend the operator-task with a new field `parameterFile`:

```yaml
tasks:
- name: deploy-bb
  kind: KudoOperator
  spec:
    parameterFile: bb-params.yaml # optional, defines the parameter that will be set on the bb-instance  
```

The contents of the `bb-params.yaml` and the top-level `AA` `params.yaml`:

```yaml
# operator-aa/templates/bb-params.yaml
# Note that I placed it under templates mostly because it also uses templating
PASSWORD: {{ .Params.BB_PASSWORD }}
```

The `PASSWORD` value is computed on the server-side when IC executes the `deploy-bb` task. The `BB_PASSWORD` parameter is defined as usual in the top-level `params.yaml` file. 

```yaml
# operator-aa/params.yaml
apiVersion: kudo.dev/v1beta1
parameters:
  - name: BB_PASSWORD
    displayName: "BB password"
    description: "password for the underlying instance of BB"
    required: true
```

This is where we see the encapsulation is action. Every operator that incorporates other operators has to define all necessary parameters at the top-level. When installing the operator `AA` the user then has to define the `BB_PASSWORD` as usual:

```bash
$ kubectl kudo install AA -p BB_PASSWORD=secret
```  

which will create `OperatorVersion-AA` 

```yaml
# /apis/kudo.dev/v1beta1/namespaces/default/operatorversions/aa-0.1.0
spec:
  operator:
    kind: Operator
    name: dummy
  parameters:
  - name: BB_PASSWORD
    displayName: "BB password"
    description: "password for the underlying instance of BB"
    required: true
  tasks:
  - name: deploy-bb
    kind: KudoOperator
    spec:
      parameterFile: bb-params.yaml
  templates:
    bb-params.yaml: |
      PASSWORD: {{ .Params.BB_PASSWORD }}
  plans:
    deploy:
      ...  
```

and `Instance-AA` resources

```yaml
# /apis/kudo.dev/v1beta1/namespaces/default/instances/instance-aa
spec:
  parameters:
    BB_PASSWORD: secret
```

During the execution of the `deploy-bb` task, the `bb-params.yaml` is expanded the same way we expand templates during the apply-task execution. The `deploy-bb` operator-task then creates the `Instance-BB` resource and saves the expanded parameter `PASSWORD: secret` in it.

What happens if we have a deeper nested operator-tasks tree e.g.:
```yaml
AA
└── BB
    └── CC
        └── DD  
            └── EE
```
and it is the low-level `EE` operator that needs the password? It is like the dependency injection through constructor parameters: every higher-level operator has to encapsulate the password parameter so that `AA` has the `BB_PASSWORD`, `BB` the `CC_PASSWORD` and so on.

 #### Update

Updating parameters work the same way as deploying the operator. In most cases, the `deploy` plan is executed. Since all dependencies already exist, the KUDO manager will traverse the dependency graph, updating Instance parameters. This will then trigger a corresponding `deploy` plan on each affected `Instance`. If the `Instance` hasn't changed no plan will be triggered.

#### Upgrade

While an out-of-band upgrade of the individual dependency operators is possible (and practically impossible to prohibit until KUDO learns drift detection), operators, in general, should be upgraded as a whole to preserve compatibility between all dependencies. An `upgrade` plan execution is very similar to the `deploy` plan. CLI creates new `OperatorVersion` resources for all new dependency operator versions. KUDO manager builds a dependency graph by traversing the `upgrade` plans of the operators and executes them in a similar fashion.

#### Uninstalling

Current `kudo uninstall` CLI command only removes instances (with required `--instance` option) using [Background deletion propagation](https://github.com/kudobuilder/kudo/blob/main/pkg/kudoctl/util/kudo/kudo.go#L281). Remember that we've added top-level `Instance` reference to the dependency operators `ownerReferences` list during [deployment](#deployment). Now we can simply delete the top-level `Instance` and let the GC delete all the others.

#### Other Plans

It can be meaningful to allow [operator-tasks](#kudooperator-task) outside of `deploy`, `update` and `upgrade` plans. A `monitoring` plan might install a monitoring operator package. We could even allow installation from a local disk by doing the same client-side steps for the `monitoring` plan when it is triggered. While the foundation provided by this KEP would make it easy, this KEP focuses on the installation dependencies, so we would probably forbid operator-tasks outside of `deploy`, `update` and `upgrade` in the beginning.

### Risks and Mitigation

The biggest risk is the increased complexity of the instance controller and the workflow engine. With the above approach, we can reuse much of the code and UX we have currently: plans and phases for flow control, local operators and custom operator repositories for easier development and deployment, and usual status reporting for debugging. The API footprint remains small as the only new API element is the [operator-task](#kudooperator-task). Dependency graph building and traversal will require a graph library and there are a [few](https://github.com/yourbasic/graph) [out](https://godoc.org/github.com/twmb/algoimpl/go/graph) [there](https://godoc.org/gonum.org/v1/gonum/graph) so this will help mitigate some complexity.

## Alternatives

One alternative is to use terraform and the existing [KUDO terraform provider](https://kudo.dev/blog/blog-2020-02-07-kudo-terraform-provider-1.html#current-process) to outsource the burden of dealing with the dependency graphs. On the upside, we would avoid the additional implementation complexity in KUDO _itself_ (though the complexity of the terraform provider is not going anywhere) and get [output values](https://www.terraform.io/docs/configuration/outputs.html) and [resource referencing](https://www.terraform.io/docs/configuration/resources.html#referring-to-instances) on top. On the downside, terraform is a heavy dependency which will completely replace KUDO UI. It is hard to quantify the pros and cons of both approaches, so it is left up for discussion.
