---
title: FAQ
type: docs
menu: docs
---

# Frequently Asked Questions

## Table of Contents

- [Frequently Asked Questions](#Frequently-Asked-Questions)
  - [Table of Contents](#Table-of-Contents)
  - [What is KUDO?](#What-is-KUDO)
  - [When would you use KUDO?](#When-would-you-use-KUDO)
  - [What is a Operator?](#What-is-a-Operator)
  - [What is a deployable service?](#What-is-a-deployable-service)
  - [What is a OperatorVersion?](#What-is-a-OperatorVersion)
  - [What is an Instance?](#What-is-an-Instance)
  - [What is a Plan?](#What-is-a-Plan)
  - [What is a PlanExecution?](#What-is-a-PlanExecution)
  - [What is a Parameter?](#What-is-a-Parameter)
  - [What is a Deployment Strategy?](#What-is-a-Deployment-Strategy)
  - [What is a Trigger?](#What-is-a-Trigger)
  - [When I create a Operator, will it automatically create new CRDs?](#When-I-create-a-Operator-will-it-automatically-create-new-CRDs)
  - [How does it work from a RBAC perspective?](#How-does-it-work-from-a-RBAC-perspective)
  - [Is the dependency model an individual controller per workload?](#Is-the-dependency-model-an-individual-controller-per-workload)

## What is KUDO?

Kubernetes Universal Declarative Operator (KUDO) provides a declarative approach to building production-grade Kubernetes [Operators](https://coreos.com/operators/). It is an operator that is specifically designed to help provide operations to operators. We want to capture the actions that are required for managing applications in production as part of the definition for the applications themselves. Further we want to embed those best practices in the operator [CRD](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

KUDO-based Operators don’t require any code in most cases, which significantly accelerates the development of Operators. It also eliminates sources of error and code duplication.

## When would you use KUDO?

You would use KUDO when `kubectl apply -f` isn't quite enough to manage your application. If you are just applying the same YAML on updates you probably won't need the extra business logic KUDO gives you.
KUDO should be used any time you would use an Operator. It can provide an advanced user experience, automating such features as updates, backups and scaling.

## What is a Operator?

`Operator` is the high-level description of a deployable service which is represented as an CRD object. An example `Operator` is the [kafka-operator.yaml](https://github.com/kudobuilder/operators/blob/master/repo/stable/kafka/versions/0/kafka-operator.yaml) that you find in the [kudobuilder/operators](https://github.com/kudobuilder/operators) repository.

Kafka Operator Example
```
apiVersion: kudo.dev/v1alpha1
kind: Operator
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: kafka
```

More examples can be found in the [https://github.com/kudobuilder/operators](https://github.com/kudobuilder/operators) project and include: [flink](https://flink.apache.org/), [kafka](https://kafka.apache.org/), and [zookeeper](https://zookeeper.apache.org/).


## What is a deployable service?

A deployable service is simply a service that is deployed on a cluster. Some services are more conceptual than that, which is what KUDO aims to help with. Cassandra for instance is a service, however, in another sense, it is a concept: a collection of data service nodes. It is the collection of these instances that make up Cassandra. A Cassandra Operator would capture the concept that you want to manage a Cassandra cluster. The OperatorVersion is the specific version of Cassandra along with any special plans to manage that cluster as outlined below.


## What is a OperatorVersion?

A `OperatorVersion` is the particular implementation of a `Operator` containing:

- [Parameters](#what-is-a-parameter)
- [Plans](#what-is-a-plan)
- [Kubernetes Objects](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/)

An example for a `OperatorVersion` is the [kafka-operatorversion.yaml](https://github.com/kudobuilder/operators/blob/master/repo/stable/kafka/versions/0/kafka-operatorversion.yaml) that you find in the [kudobuilder/operators](https://github.com/kudobuilder/operators) repository.

```
apiVersion: kudo.dev/v1alpha1
kind: OperatorVersion
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: kafka-2.11-2.4.0
  namespace: default
spec:
  serviceSpec:
  version: "2.11-2.4.0"
  connectionString: ""
  operator:
    name: kafka
    kind: Operator
  parameters:
    - name: BROKER_COUNT
      description: "Number of brokers spun up for Kafka"
      default: "3"
      displayName: "Broker Count"
      ...
plans:
   deploy:
     strategy: serial
     phases:
       - name: deploy-kafka
         strategy: serial
         steps:
           - name: deploy
             tasks:
               - deploy      
```

The purpose of the OperatorVersion is to provide the details necessary for KUDO to become an operator for a specific capability (such as Kafka) for a version of the operator. As the operator it will execute a `Plan` to establish all the instances of Kakfa components in the cluster as defined in the Operator yaml. In the example provided, it would make sure there are 3 brokers deployed.

## What is an Instance?

An `Instance` is a CRD object used as *linker* which ties an application instantiation to a `OperatorVersion`.

## What is a Plan?

Plans are how KUDO operators convey progress through service management operations, such as repairing failed tasks and/or rolling out changes to the service’s configuration. Each Plan is a tree with a fixed three-level hierarchy of the Plan itself, its Phases, and Steps within those Phases. These are all collectively referred to as “Elements”. The fixed tree hierarchy was chosen in order to simplify building user interfaces that display plan content. This three-level hierarchy can look as follows:

```bash
Plan foo
├─ Phase bar
│  ├─ Step qux
│  └─ Step quux
└─ Phase baz
   ├─ Step quuz
   ├─ Step corge
   └─ Step grault
```

The status of the execution of a Plan is captured as a [PlanExecution](#what-is-a-planexecution) CRD. Each execution of a Plan has intentionally its own `PlanExecution` CRD. Plans drive the transition between current and desired states, and are built based on the current progress of that transition. They are not themselves part of those states.

KUDO expects by default the `deploy` Plan.

## What is a PlanExecution?

This CRD captures the status of the execution of a Plan defined in a `OperatorVersion` on an `Instance`. The Plan status is solely determined based on the statuses of its child Phases, and the Phases in turn determine their statuses based on their Steps.

## What is a Parameter?

Parameters provide configuration for the Instance.

## What is a Deployment Strategy?

A deployments strategy indicates the way in which a Plan or Step must be executed. If a Step requires another Step to complete first, it is necessary to declare them as `serial`. The following strategies are available by default and can be used in a `OperatorVersion` YAML definition:

- `serial`
  An example for a `serial` Plan is [kafka-operatorversion.yaml](https://github.com/kudobuilder/operators/blob/master/repo/stable/kafka/versions/0/kafka-operatorversion.yaml).
- `parallel`
  An example for a `parallel` Plan is [zookeeper-operatorversion.yaml](https://github.com/kudobuilder/operators/blob/master/repo/stable/kafka/versions/0/zookeeper-operatorversion.yaml)`

## What is a Trigger?

When a Parameter is updated in an `Instance` object, it defines the "update strategy" for the Parameter in the `OperatorVersion`. This also gives you the option to customize your Plan a little bit further. For instance, a change of a specific value can trigger a pre-defined Plan, for example the `update` plan. You can define distinct update strategies for different Parameters. For example, you might trigger the `update` Plan when `image` is changed, and another Plan when `replicas` is changed.

## When I create a Operator, will it automatically create new CRDs?

That is the eventual goal. We want each OperatorVersion (those versions of a Operator) to be a different API version of a command CRD that gets mapped to the Operator (see also image below). A Operator creates a CRD and then the versions of those are defined by the OperatorVersion.

![KUDO dynamic CRD](../images/kudo-dymanic-crd.png?10x20)

## How does it work from a RBAC perspective?

Right now everything is `namespaced`. For the current capability `Operator`, `OperatorVersion`,`Instance` and `PlanExecution` are all namespaced and the permissions of the operator are all in the current namespace. For example, deletion can only happen on objects in the namespace that the instance is in. There is a trade-off between the *flexibility* of having application operators deploy their own versions in their own namespaces to manage versus having *broad capability* from a cluster perspective. With easily defining `OperatorVersions` in YAML we give you the capability to provide full operators to everyone on the cluster and you are able to give those application management pieces to those application operators individually and not have to provide support on each one of those.

## Is the dependency model an individual controller per workload?

We have one controller that handles all of the CRDs of the `Operator`, `OperatorVersion`, `Instance` and `PlanExecution` types. They all are being subscribed by the same single state machine operator. For example, right now there is only an `Instance` CRD and that object is owned by its single operator. Once we start doing dynamic CRDs there will be more types dynamically subscribed by new objects registering with the operator along the way.
