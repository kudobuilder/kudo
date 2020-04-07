---
kep-number: 29
title: Operator Dependencies
short-desc: Introducing operators depending on other operators
authors:
  - "@zen-dog"
owners:
  - "@zen-dog"
editor: @zen-dog
creation-date: 2020-03-30
last-updated: 2020-03-30
status: provisional
---

# Operator Dependencies

## Table of Contents

* [Summary](#summary)
* [Motivation](#motivation)
  * [Goals](#goals)
  * [Non-Goals](#non-goals)
* [Proposal](#proposal)
  * [Implementation Details](#implementation-detailsnotesconstraints)
    * [Operator Task](#operator-task)
    * [Deployment](#deployment)
      * [Client-Side](#client-side)
      * [Server-Side](#server-side)
      * [Parameterization](#dependencies-parameterization)
    * [Update](#update)
    * [Upgrade](#upgrade)
    * [Uninstalling](#uninstalling)
  * [Risks and Mitigation](#risks-and-mitigation)
* [Alternatives](#alternatives)

## Summary

This KEP aims to improve operator user and developer experience by introducing operator dependencies.

## Motivation

Recent operator development has shown that complex operators often depend on other operators to function. One workaround commonly seen can be found in the [Flink Demo](https://github.com/kudobuilder/operators/tree/master/repository/flink/docs/demo/financial-fraud). It requires manual installation of the dependency operators while skipping the instance creation (using the `--skip-instance` option). The `Instance` is then "manually" created during the execution of the `deploy` plan. This KEP is aiming at improving this experience, streamlining the developer and user experience.

### Goals

Dependencies can be a complex topic. This KEP is not trying to boil the dependency ocean but rather limits itself to installation dependencies only.

### Non-Goals

Dependency on an already running `Instance` is a non-goal. It is easy to imagine a situation when a new operator (e.g Kafka) may want to depend on the existing Zookeeper instance. However, such life-cycle dependency presents major challenges e.g. what happens when Zookeeper is removed? What happens when Zookeeper is upgraded and the new version is incompatible with the current Kafka `Instance`? How can compatibility be ensured? This KEP deliberately ignores this area and instead focuses on installation dependencies. Additionally, this KEP does not address output variables or referencing `Instance` resources.

## Proposal

KUDO operators **already have** a mechanism to deal with installation dependencies called [plans, phases, and steps](https://kudo.dev/docs/developing-operators/plans.html#overview) with serial or parallel execution strategy. This mechanism is already powerful enough to express any dependency hierarchy including transitive dependencies (see more about this in [implementation details](#implementation-details)). The core of this proposal is to reuse this mechanism and extend it with the ability to install operators.

### Implementation details

#### Operator Task

A new task kind `Operator` is introduced which extends `operator.yaml` with the ability to install dependencies. Let's take a look at the Kafka+Zookeeper example:

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
          - name: dummy
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
  kind: Operator
  spec:
    package: # required, either repo package name, local package folder or an URL to package tarball
    repo: # optional, name of local repository configuration to use
    appVersion: # optional, a specific app version in the official repo, defaults to the most recent one
    operatorVersion: # optional, a specific operator version in the official repo, defaults to the most recent one
    instanceName: # optional, the instance name
```

As you can see, this closely mimics the `kudo install` CLI command [options](https://github.com/kudobuilder/kudo/blob/master/pkg/kudoctl/cmd/install.go#L56) because at the end the later will be executed to install the operator. We omit `parameters` and `parameterFile` options at this point as they are discussed in detail [below](#dependencies-parameterization)

#### Deployment

##### Client-Side

Upon execution of `kudo install ...` command with the above operator definition, CLI will:

1. Collect all operator dependencies by analyzing the `deploy` plan of the top-level operator and compose a list of all operator dependencies (tasks with the kind `Operator`) including **transitive dependencies**
2. Install all collected operators **skipping the instances** (same as `kudo install ... --skip-instance`). Note that since we do not create instances here we can install them in any order
3. Proceed with the installation of the top-level operator as usual (create `Operator`, `OperatorVersion` and `Instance` resources)

Since we do this step on the client-side we have access to the full functionality of the `install` command including installing operators from the file system. This will come very handy during the development and debugging which arguably becomes more complex with dependencies.

##### Server-Side

Upon receiving a new operator Instance with dependencies KUDO mangers workflow engine will:

1. Build a [dependency graph](https://en.wikipedia.org/wiki/Dependency_graph) by transitively expanding top-level `deploy` plan using operator-tasks as vertices and their execution order (`a` needs `b` to be installed first) as edges
2. Perform cycle detection and fail if circular dependencies found. We could additionally run this check on the client-side as part of the `kudo package verify` command to improve the UX
3. If no cycles were found we traverse the dependency graph in the topological order (e.g. using the reversed post-order) and execute all vertices

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

In the first step we build a dependency graph. A set of all graph vertices (which are task-operators) `S` is defined as `S = {AA, BB, CC, EE, GG}`. A transitive relationship `R` between the vertices is defined as `(a, b) ∈ S` meaning that _`a` needs `b` deployed first_. The transitive relationship for the above example is: `R = { (AA,BB), (AA,CC), (BB,EE), (BB,GG) }`. The resulting topological order `O` is therefor `O = (EE, GG, BB, CC, AA)` which has no cycles.

The instance controller (IC) then traverses the dependency graph in the evaluation order `O` and executes each vertex. Practically this means creating a corresponding `Instance` resource (`Operator` and `OperatorVersion` already exist) and wait for it to become healthy, meaning that its `deploy` plan was executed successfully. KUDO manager already has [health check](https://github.com/kudobuilder/kudo/blob/master/pkg/engine/health/health.go#L78) for `Instance` resources implemented. We would additionally add the top-level `Instance` reference (e.g. `AA`) to the `ownerReferences` list of each dependency `Instance` (e.g. `BB`). This would help with determining which `Instance` belongs to which operator and additionally help us with [operator uninstalling](#uninstalling).

The status of the execution can be seen as usual as part of the `Instance.Status`. We could additionally forward the status of a dependency `Instance` to the top-level `Instance.Status` to simplify the overview.

Note that in the above example could not depend on the same `Instance` as it will create a dependency cycle. So if e.g. `EE` and `CC` are instances of the same `OperatorVersion`, they must use distinct names and will be installed as two separate `Instances`.

##### Dependencies Parameterization

First, we need to provide parameters to different operators separately. For individual parameters (`-p` option) we can use namespaced names e.g. `-p <instanceName>.<key>=<value>`. For the parameter files (`--parameterFile` option) we could extend `parameterFile` schema (which is currently a simple map) with top-level fields e.g.:

```yaml
apiVersion: v1beta1
instance: zookeeper-instance
parameters:
  foo: bar # other parameterFile keys and values
```

Second, parameters are normally parsed to the CLI and persisted in the `Instance` resource. Since the creation of the instances is delayed we need a place to store the parameters until then. We could keep them in the task definition e.g.:

```yaml
tasks:
- name: demo
  kind: Operator
  spec:
    ...
    parameters: # optional, a map of parameter keys and values constructed by expanding the parameterFiles along with individual parameters

```

KUDO manager would then read them and populate `Instance.Spec.Parameters` as usual. The drawback here is that this would likely stretch currently slim task definitions.
Alternatively, we could introduce a new field e.g. `OperatorVersionSpec.InstallationParameters` alongside with the existing [Parameters](https://github.com/kudobuilder/kudo/blob/master/pkg/apis/kudo/v1beta1/operatorversion_types.go#L35). Both approaches contradict our current data model where such parameters would exist in the `Instance` resource.

#### Update

Updating parameters work the same way as deploying the operator. In most cases, the `deploy` plan is executed. Since all dependencies already exist, the KUDO manager will traverse the dependency graph, updating Instance parameters. This will then trigger a corresponding `deploy` plan on each affected `Instance`. If the `Instance` hasn't changed no plan will be triggered.

#### Upgrade

While an out-of-band upgrade of the individual dependency operators is possible (and practically impossible to prohibit), operators, in general, should be upgraded as a whole to preserve compatibility between all dependencies. An `upgrade` plan execution is very similar to the `deploy` plan. CLI creates new `OperatorVersion` resources for all new dependency operator versions. KUDO manager builds a dependency graph by traversing the `upgrade` plans of the operators and executes them in a similar fashion.

#### Uninstalling

Current `kudo uninstall` CLI command only removes instances (with required `--instance` option) using [Background deletion propagation](https://github.com/kudobuilder/kudo/blob/master/pkg/kudoctl/util/kudo/kudo.go#L281). Remember that we've added top-level `Instance` reference to the dependency operators `ownerReferences` list during [deployment](#deployment). Now we can simply delete the top-level `Instance` and let the GC delete all the others.

### Risks and Mitigation

The biggest risk is the increased complexity of the instance controller and the workflow engine. With the above approach, we can reuse much of the code and UX we have currently: plans and phases for flow control, local operators and custom operator repositories for easier development and deployment, and usual status reporting for debugging. The API footprint remains small as the only new API element is the [operator-task](#operator-task). Dependency graph building and traversal will require a graph library and there are a [few](https://github.com/yourbasic/graph) [out](https://godoc.org/github.com/twmb/algoimpl/go/graph) [there](https://godoc.org/gonum.org/v1/gonum/graph) so this will help mitigate some of the complexity.

## Alternatives

One alternative is to use terraform and the existing [KUDO terraform provider](https://kudo.dev/blog/blog-2020-02-07-kudo-terraform-provider-1.html#current-process) to outsource the burden of dealing with the dependency graphs. On the upside, we would avoid the additional implementation complexity in KUDO _itself_ (though the complexity of the terraform provider is not going anywhere) and get output variables and referencing resources on top. On the downside, this would tie terraform to KUDOs metaphorical leg. It is hard to quantify the pros and cons of both approaches so it is left up for discussion.
