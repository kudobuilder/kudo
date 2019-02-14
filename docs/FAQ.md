### Frequently Asked Questions

#### Table of Contents
* [What is KUDO?](#what-is-kudo)
* [When would you use KUDO?](#when-would-you-use-kudo)
* [What is a Framework](#what-is-a-framework)
* [What is a Frameworkversion?](#what-is-a-frameworkversion)
* [What is an Instance?](#what-is-an-instance)
* [What is a Planexecution?](#what-is-a-planexecution)
* [What is a Parameter?](#what-is-a-parameter)
* [What is a Trigger?](#what-is-a-trigger)
* [When I create a Framework, it is automatically creating new CRDs?](#when-i-create-a-framework-it-is-automatically-creating-new-crds)
* [How does it work from a RBAC perspective?](#how-does-it-work-from-a-rbac-perspective)
* [Is the dependency model an individual controller per workload?](#is-the-dependency-model-an-individual-controller-per-workload)

#### What is KUDO?

KUDO is an operator we are building that is specifically designed to help provide operations to operators. We want to capture the actions that are required for managing applications in production as part of the deployment for the applications themself. To embed those best practices in the operator CRD. So KUDO is the operator that handles those specs that are provided as declarative CRDs.

#### When would you use KUDO?

You would use KUDO when `kubectl apply -f` isn't quite enough to manage your application. If you are just applying the same YAML on updates you probably won't need the extra business logic KUDO gives you.

#### What is a Framework

`Framework` is the highlevel CRD object.

#### What is a Frameworkversion?

#### What is an Instance?

`Instance` is a CRD object used as *linker* which ties an application instantiation to a `Frameworkversion`.

#### What is a Planexecution?

#### What is a Parameter?

#### What is a Trigger?

When a value is updated in an `Instance` object it defines which plan should run. This gives you also the option of customizing your plans a little bit further, e.g. that a change of a specific value triggers a pre-defined plan, for example the `deploy` plan. 

#### When I create a Framework, it is automatically creating new CRDs?

That is the eventual goal. We want each frameworkversion (on of those versions of a framework) to be a different API version of a commond CRD that gets mapped to the framework (see also [here](https://docs.google.com/presentation/d/1ZoepKFbv7HTBbwww2DGufJhxLbCif4xv9jYyg_oJ54k/edit#slide=id.g4f6d9f5d82_0_53)). A Framework creates a CRD and then the versions of those are defined by the Frameworkversion.

##### How does it work from a RBAC perspective?

Right now everything is `namespaced`. For the current capability `Framework`, `Frameworkversion`,`Instance` and `PlanExecution` are all namespaced and the permissions of the operator are all in the current namespace. For example, deletion can only happen on objects in the namespace that the instance is in. There is a trade-off between the *flexibility* of having application operators deploy their own versions in their own namespaces to manage versus having *broad capability* from a cluster perspective. With easily defining `Frameworkversions` in YAML we give you the capbability to provide full operators to everyone on the cluster and you are able to give those application management pieces to those application operators individually and not have to provide support on each one of those.

#### Is the dependency model an individual controller per workload?

We have one controller that handles all of the CRDs of the `Framework`, `Frameworkversion`, `Instance` and `Planexecution` types. They all are being subscribed by the same single state machine operator. For example, right now there is only an `Instance` CRD and that object is owned by its single operator. Once we start doing dynamic CRDs there will be more types dynamically subscribed by new objects registering with the operator along the way.