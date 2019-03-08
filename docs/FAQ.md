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

KUDO is an operator we are building that is specifically designed to help provide operations to operators. We want to capture the actions that are required for managing applications in production as part of the definition for the applications themself. Further we want to embed those best practices in the operator CRD. So KUDO is the operator that handles those specs that are provided as declarative CRDs.

#### When would you use KUDO?

You would use KUDO when `kubectl apply -f` isn't quite enough to manage your application. If you are just applying the same YAML on updates you probably won't need the extra business logic KUDO gives you.

#### What is a Framework

`Framework` is the highlevel CRD object. An example for a `Framework` is e.g. the [kafka-framework.yaml](https://github.com/kudobuilder/frameworks/blob/master/repo/stable/kafka/versions/0/kafka-framework.yaml) that you find in the [kudobuilder/frameworks](https://github.com/kudobuilder/frameworks) repo.

#### What is a FrameworkVersion?

A `FrameworkVersion` is the particular implementation of a `Framework`. It contains: 

- [Parameters](#what-is-a-parameter)
- [Plans](#what-is-a-plan)
- Kubernetes Objects

An example for a `FrameworkVersion` is e.g. the [kafka-frameworkversion.yaml](https://github.com/kudobuilder/frameworks/blob/master/repo/stable/kafka/versions/0/kafka-frameworkversion.yaml) that you find in the [kudobuilder/frameworks](https://github.com/kudobuilder/frameworks) repo.

#### What is an Instance?

`Instance` is a CRD object used as *linker* which ties an application instantiation to a `FrameworkVersion`.

#### What is a Plan?

Plans are how KUDO frameworks convey progress through service management operations, such as repairing failed tasks and/or rolling out changes to the service’s configuration. Each Plan is a tree with a fixed three-level hierarchy of the Plan itself, its Phases, and then Steps within those Phases. These are all collectively referred to as “Elements”. The choice of three levels was arbitrarily chosen as “enough levels for anybody”. The fixed tree hierarchy was chosen in order to simplify building UIs that display plan content. In particular, lots of suggestions were made to have a full DAG structure, which were ultimately rejected. This three-level hierarchy can look as follows:

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

The status of the execution of a Plan is captured an extra [PlanExecution](#what-is-a-planexecution) CRD. Each execution of a Plan has intentional its own `PlanExecution` CRD. Plans drive the transition between current and desired states, and are built based on the current progress of that transition. They are not themselves part of those states.

KUDO expects by default the `deploy` Plan.

#### What is a PlanExecution?

This CRD captures the status of the execution of a Plan defined in a `FrameworkVersion` on an `Instance`. The Plan status is solely determined based on the statuses of its child Phases, and the Phases in turn determine their statuses based on their Steps.

#### What is a Parameter?

Parameters provide configuration for the Instance.

#### What is a Strategy?

The following strategies are available by default, these can be used in a `FrameworkVersion` YAML definition:

- `serial` and `parallel`

An example for a `serial` Plan is [kafka-frameworkversion.yaml](https://github.com/kudobuilder/frameworks/blob/master/repo/stable/kafka/versions/0/kafka-frameworkversion.yaml).
An example for a `parallel` Plan is [zookeeper-frameworkversion.yaml](https://github.com/kudobuilder/frameworks/blob/master/repo/stable/kafka/versions/0/zookeeper-frameworkversion.yaml)`

#### What is a Trigger?

When a Parameter is updated in an `Instance` object it defines the "update strategy" for the Parameter in the `FrameworkVersion`. This gives you also the option to customize your Plan a little bit further, e.g. that a change of a specific value triggers a pre-defined Plan, for example the `update` plan. It also allows different update strategies for different Parameters (e.g. `replicas` vs `image`)

#### When I create a Framework, it is automatically creating new CRDs?

That is the eventual goal. We want each frameworkversion (on of those versions of a framework) to be a different API version of a commond CRD that gets mapped to the framework (see also [here](https://docs.google.com/presentation/d/1ZoepKFbv7HTBbwww2DGufJhxLbCif4xv9jYyg_oJ54k/edit#slide=id.g4f6d9f5d82_0_53)). A Framework creates a CRD and then the versions of those are defined by the Frameworkversion.

##### How does it work from a RBAC perspective?

Right now everything is `namespaced`. For the current capability `Framework`, `Frameworkversion`,`Instance` and `PlanExecution` are all namespaced and the permissions of the operator are all in the current namespace. For example, deletion can only happen on objects in the namespace that the instance is in. There is a trade-off between the *flexibility* of having application operators deploy their own versions in their own namespaces to manage versus having *broad capability* from a cluster perspective. With easily defining `Frameworkversions` in YAML we give you the capbability to provide full operators to everyone on the cluster and you are able to give those application management pieces to those application operators individually and not have to provide support on each one of those.

#### Is the dependency model an individual controller per workload?

We have one controller that handles all of the CRDs of the `Framework`, `Frameworkversion`, `Instance` and `Planexecution` types. They all are being subscribed by the same single state machine operator. For example, right now there is only an `Instance` CRD and that object is owned by its single operator. Once we start doing dynamic CRDs there will be more types dynamically subscribed by new objects registering with the operator along the way.