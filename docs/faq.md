---
title: FAQ
type: docs
weight: 7
---

### Frequently Asked Questions

#### Table of Contents

         * [Frequently Asked Questions](#frequently-asked-questions)
            * [Table of Contents](#table-of-contents)
            * [What is KUDO?](#what-is-kudo)
            * [When would you use KUDO?](#when-would-you-use-kudo)
            * [What is a Framework](#what-is-a-framework)
            * [What is a FrameworkVersion?](#what-is-a-frameworkversion)
            * [What is an Instance?](#what-is-an-instance)
            * [What is a Plan?](#what-is-a-plan)
            * [What is a PlanExecution?](#what-is-a-planexecution)
            * [What is a Parameter?](#what-is-a-parameter)
            * [What is a Strategy?](#what-is-a-strategy)
            * [What is a Trigger?](#what-is-a-trigger)
            * [When I create a Framework, it is automatically creating new CRDs?](#when-i-create-a-framework-it-is-automatically-creating-new-crds)
               * [How does it work from a RBAC perspective?](#how-does-it-work-from-a-rbac-perspective)
            * [Is the dependency model an individual controller per workload?](#is-the-dependency-model-an-individual-controller-per-workload)

#### What is KUDO?

Kubernetes Universal Declarative Operator (KUDO) provides a declarative approach to building production-grade Kubernetes [Operators](https://coreos.com/operators/). It is an operator that is specifically designed to help provide operations to operators. We want to capture the actions that are required for managing applications in production as part of the definition for the applications themself. Further we want to embed those best practices in the operator [CRD](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

KUDO-based Operators don’t require any code in most cases, which significantly accelerates the development of Operators. It also eliminates sources of error and code duplication.

#### When would you use KUDO?

You would use KUDO when `kubectl apply -f` isn't quite enough to manage your application. If you are just applying the same YAML on updates you probably won't need the extra business logic KUDO gives you.
KUDO should be used any time you would use an Operator. It can provide an advanced user experience, automating such features as updates, backups and scaling.

#### What is a Framework

`Framework` is the high-level description of a deployable service which is represented as an CRD object. An example `Framework` is the [kafka-framework.yaml](https://github.com/kudobuilder/frameworks/blob/master/repo/stable/kafka/versions/0/kafka-framework.yaml) that you find in the [kudobuilder/frameworks](https://github.com/kudobuilder/frameworks) repository.

Kafka Framework Example
```
apiVersion: kudo.k8s.io/v1alpha1
kind: Framework
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: kafka
```

More examples can be found in the [https://github.com/kudobuilder/frameworks](https://github.com/kudobuilder/frameworks) project and include: [flink](https://flink.apache.org/), [kafka](https://kafka.apache.org/), and [zookeeper](https://zookeeper.apache.org/).


#### What is a deployable service?

Deployable service is simple a service that is deployed on a cluster, however some services are more conceptual which is what KUDO aims to help with. For instance, Cassandra is a service, however in another since it is a concept. It really is a collection of data service nodes. It is the collection of these instances that make of Cassandra. A Cassandra Framework would capture the concept that you want to manage a Cassandra cluster. The FrameworkVersion is the specific version of Cassandra along with any special plans to manage that cluster as outlined below.


#### What is a FrameworkVersion?

A `FrameworkVersion` is the particular implementation of a `Framework` containing:

- [Parameters](#what-is-a-parameter)
- [Plans](#what-is-a-plan)
- [Kubernetes Objects](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/)

An example for a `FrameworkVersion` is the [kafka-frameworkversion.yaml](https://github.com/kudobuilder/frameworks/blob/master/repo/stable/kafka/versions/0/kafka-frameworkversion.yaml) that you find in the [kudobuilder/frameworks](https://github.com/kudobuilder/frameworks) repository.

```
apiVersion: kudo.k8s.io/v1alpha1
kind: FrameworkVersion
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: kafka-2.11-2.4.0
  namespace: default
spec:
  serviceSpec:
  version: "2.11-2.4.0"
  connectionString: ""
  framework:
    name: kafka
    kind: Framework
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

The purpose of the FrameworkVersion is to provide the details necessary for KUDO to become an operator for a specific capability (such as Kafka) for a version of the framework. As the operator it will execute a `Plan` to establish an all the instances of Kakfa components in the cluster as detailed in the Framework yaml. In the example provided it would make sure there are 3 brokers for instance.

#### What is an Instance?

`Instance` is a CRD object used as *linker* which ties an application instantiation to a `FrameworkVersion`.

#### What is a Plan?

Plans are how KUDO frameworks convey progress through service management operations, such as repairing failed tasks and/or rolling out changes to the service’s configuration. Each Plan is a tree with a fixed three-level hierarchy of the Plan itself, its Phases, and then Steps within those Phases. These are all collectively referred to as “Elements”. The fixed tree hierarchy was chosen in order to simplify building UIs that display plan content. This three-level hierarchy can look as follows:

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

The status of the execution of a Plan is captured an [PlanExecution](#what-is-a-planexecution) CRD. Each execution of a Plan has intentionally its own `PlanExecution` CRD. Plans drive the transition between current and desired states, and are built based on the current progress of that transition. They are not themselves part of those states.

KUDO expects by default the `deploy` Plan.

#### What is a PlanExecution?

This CRD captures the status of the execution of a Plan defined in a `FrameworkVersion` on an `Instance`. The Plan status is solely determined based on the statuses of its child Phases, and the Phases in turn determine their statuses based on their Steps.

#### What is a Parameter?

Parameters provide configuration for the Instance.

#### What is a Deployment Strategy?

A deployments strategy indicates the way in which a plan, or Step must be executed. If a Step requires another Step to complete first it is necessary to declare them as `serial`. The following strategies are available by default, these can be used in a `FrameworkVersion` YAML definition:

- `serial` and `parallel`

An example for a `serial` Plan is [kafka-frameworkversion.yaml](https://github.com/kudobuilder/frameworks/blob/master/repo/stable/kafka/versions/0/kafka-frameworkversion.yaml).
An example for a `parallel` Plan is [zookeeper-frameworkversion.yaml](https://github.com/kudobuilder/frameworks/blob/master/repo/stable/kafka/versions/0/zookeeper-frameworkversion.yaml)`

#### What is a Trigger?

When a Parameter is updated in an `Instance` object it defines the "update strategy" for the Parameter in the `FrameworkVersion`. This gives you also the option to customize your Plan a little bit further, for instance a change of a specific value triggers a pre-defined Plan, for example the `update` plan. It also allows different update strategies for different Parameters (e.g. `replicas` vs `image`)

#### When I create a Framework, will it automatically create new CRDs?

That is the eventual goal. We want each FrameworkVersion (those versions of a Framework) to be a different API version of a command CRD that gets mapped to the Framework (see also image below). A Framework creates a CRD and then the versions of those are defined by the FrameworkVersion.

![Quick Start](images/kudo-dymanic-crd.png)

##### How does it work from a RBAC perspective?

Right now everything is `namespaced`. For the current capability `Framework`, `FrameworkVersion`,`Instance` and `PlanExecution` are all namespaced and the permissions of the operator are all in the current namespace. For example, deletion can only happen on objects in the namespace that the instance is in. There is a trade-off between the *flexibility* of having application operators deploy their own versions in their own namespaces to manage versus having *broad capability* from a cluster perspective. With easily defining `FrameworkVersions` in YAML we give you the capability to provide full operators to everyone on the cluster and you are able to give those application management pieces to those application operators individually and not have to provide support on each one of those.

#### Is the dependency model an individual controller per workload?

We have one controller that handles all of the CRDs of the `Framework`, `FrameworkVersion`, `Instance` and `PlanExecution` types. They all are being subscribed by the same single state machine operator. For example, right now there is only an `Instance` CRD and that object is owned by its single operator. Once we start doing dynamic CRDs there will be more types dynamically subscribed by new objects registering with the operator along the way.
