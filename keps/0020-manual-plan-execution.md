---
kep-number: 20
title: Manual plan execution
authors:
  - "@alenkacz"
  - "@zen-dog"
owners:
  - "@alenkacz"
  - "@zen-dog"
creation-date: 2019-11-08
last-updated: 2020-03-10
status: implemented
see-also:
  - KEP-18
---

# manual-plan-execution

## Table of Contents

   * [manual-plan-execution](#manual-plan-execution)
      * [Table of Contents](#table-of-contents)
      * [Summary](#summary)
      * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals)
      * [Proposal](#proposal)
         * [Update the status of Instance with name of the plan you want to run, let controller pick it up](#update-the-status-of-instance-with-name-of-the-plan-you-want-to-run-let-controller-pick-it-up)
            * [Pros](#pros)
            * [Cons](#cons)
         * [Expose custom HTTP API from controller-manager to allow receiving commands](#expose-custom-http-api-from-controller-manager-to-allow-receiving-commands)
            * [Pros](#pros-1)
            * [Cons](#cons-1)
         * [Have a separate CRD just for maintaining commands to execute a plan](#have-a-separate-crd-just-for-maintaining-commands-to-execute-a-plan)
            * [Pros](#pros-2)
            * [Cons](#cons-2)
         * [Introduce new sub-resource for triggering plan executions](#introduce-new-sub-resource-for-triggering-plan-executions)
            * [Pros](#pros-3)
            * [Cons](#cons-3)
         * [Introducing a new field in the Instance.Spec](#introducing-a-new-field-in-the-instancespec)
            * [Pros](#pros-4)
            * [Cons](#cons-4)
         * [User Stories](#user-stories)
         * [The Solution](#the-solution)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Summary

Since the inception of KUDO, being able to manually run a plan was one of the core features. After KEP-18 (controller overhaul) we changed the way we store information about executed plans practically removing the ability to execute custom plan in an easy way through kubectl. Goal of this KEP is to propose a way how to re-introduce that functionality and provide a good user experience.

## Motivation

### Goals

Operator will be able to trigger plan of given name through CLI
Controller won’t allow multiple plans running in parallel

### Non-Goals

Provide a history of plan executions older that the one last run per plan.

## Proposal

There are multiple ways we can do this. The goal of the KEP is to choose the one with the right balance of added complexity and with a good enough UX. The main challenge with executing plans is that kubernetes API in its essence is declarative while starting a plan is an imperative action (a command) so it does not fit very nicely into the ecosystem.

### Update the status of Instance with name of the plan you want to run, let controller pick it up

This is by far the simplest solution but probably not the one with cleanest design. The implementation of this could be as simple as introducing new CLI command that would do:

```go
newStatus := instance.Status.DeepCopy()
// TODO planStatus now contains status of that plan we want to run
// for implementation details of that look at instance.StartPlanExecution
newStatus.Status = ExecutionPending
newStatus.UID = uuid.NewUUID()

instance.Status = newStatus
client.Status().Update(context.TODO(), instance)
```

This also counts with the fact that on the server-side we have an admission webhook that won't allow setting a status like that in case there is another plan running. Such update would be rejected.

Although it’s very easy to do this, it’s not very Kubernetes idiomatic way of doing things especially because Status should never be updated from a client and it should just capture the state of the object.

For some background, including definition of status sub-resource by [API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status): `The status summarizes the current state of the object in the system, and is usually persisted with the object by an automated processes but may be generated on the fly. At some cost and perhaps some temporary degradation in behavior, the status could be reconstructed by observation if it were lost.`  

#### Pros

- Easy to implement considering the current state of the code

#### Cons

- No way to start a plan with just pure kubectl - this is a reply from sig-cli channel where I posed that question (`It doesn't make sense to update the status manually, it should reflect the current state in the cluster. So no, you cannot set it though kubectl`)

### Expose custom HTTP API from controller-manager to allow receiving commands

Manager binary with controllers (that is run within the pod in the cluster) would in this case have to also expose HTTP API and we’ll have to expose a kubernetes service with this custom API. All clients (CLI/user scripts) will talk to this API every time they want to start a new plan. The API could be as simple as 

HTTP POST `/start?planName=deploy`

The validation whether a plan can be started or not would have to be part of the HTTP endpoint implementation.

#### Pros

- The complexity of the implementation is still pretty manageable and lower than having a full-featured sub-resource

#### Cons

- Have to maintain a separate API (with API documentation)
- It is a non-kubernetes API, so no clients for various languages and no kubectl support (although if we keep the API that simple, almost a CURL can do the trick)
- More complex manager pod deployment with additional service
- How do we manage RBACs?

### Have a separate CRD just for maintaining commands to execute a plan

This would basically mean going half-way back to state pre KEP-18 where we had a separate CR that was created for every plan that was executed. In this implementation we could have a much leaner CRD capturing only the requests to execute a plan with very lean status that does not duplicate the instance's status. Could be as simple as:

```
apiVersion: kudo.dev/v1beta1
kind: PlanExecutionRequest
Spec:
  planName: deploy
  Instance: # this will be OwnerReference
    Name: my-instance
    Namespace: some-namespace
  Parameters:
    some-instance-param: my override
Status:
  Status: Accepted # one of Accepted, Rejected, Cancelled (added in the future)
```

`PlanExecutionRequest` will also contain status sub-resource but it will be a status reflecting whether the request was accepted or not, it should in no way be a duplication of `Instance` status as that would move us to pre-KEP-18 situation where updating status in two separate CRDs caused some inconsistencies as that cannot be done in a transactional way.

But at the same time this seems to be the way people are leaning toward when coming to similar use-cases. The discussion under [this issue](https://github.com/kubernetes/kubernetes/issues/72637#issuecomment-515566586) is kind of touching on the topic of Request objects. What this [KEP](https://github.com/metal3-io/metal3-docs/pull/48/files) proposes for reboot is also kind of similar to what we’re trying to do. Alongside what they’re proposing we might not even need a controller for this CR and we could treat it as “queue” meaning every request will be fulfilled so e.g. when you create a request and a plan is running, instance will pick up the next request when it has a capacity to run another plan.

The implementation will have following properties:

- there is no controller for `PlanExecutionRequest`s
- you cannot tell status of the execution just from the `PlanExecutionRequest` object
- `PlanExecutionRequest` is managed by the `InstanceController`. We might keep N last PERs for audit purposes, and delete old PERs eventually. All PERs will be GCed on Instance deletion.

#### Pros

- Kubernetes-native way of API for executing plans, can be easily done via kubectl or any other kubernetes client
- Seems to currently be the de-facto pattern to do these things before sub-resources via webhooks are introduced for CRDs.

#### Cons

General:
- Challenging to communicate back to the user if the execution was possible and how it happened (would need good support for this in CLI)
- Maintaining another CRD
- Garbage collection of the Request objects

Have to maintain a request queue. This is trickier than it looks at first glance: 
- having a Request sequence at all is counter-intuitive: e.g. an operator will have to check for what's already in the queue, before scheduling a new Request. If the current plan is stuck in execution (due to an error, insufficient cluster resources or otherwise) an operator would likely want to correct the existing plan, instead of scheduling a new one. This will require an additional layer of API complexity (scheduling call vs. overriding a current one)
- not all Request sequences are meaningful, e.g. an Upgrade followed by a Request to a non-existing, renamed or otherwise outdated plan can't be executed without any meaningful way of telling the user why the Request was not handled (except for logs or events)

Cross-resource consistency:  
This is a fundamental limitation of Kubernetes (or rather it is inherited from etcd). if a controller has to read/write two resources to make a decision, there is always a room for inconsistencies. And while there are strategies of dealing with said inconsistencies, most of which fall into the "eventually-consistent" category, this is still not trivial. While, on the one hand, an Instance can always be updated (triggering plans) and upgraded directly a user can always manually create Requests in parallel. There is always an ambiguity of what is handled first and a problem of no immediate feedback to the user. And webhooks can not help here, because a webhook can only meaningfully guard **one** resource and not multiple.

**tl;dr**: We believe that a queue is not a meaningful paradigm when working with data services (or other complex applications). Each plan execution has a big failure potential and scheduling a series of such Requests complicate dealing with systems unnecessary. And dealing with cross-resource inconsistencies and lack of immediate feedback just add an insult to misery.

### Introduce new sub-resource for triggering plan executions

Another solution for this problem would Kubernetes support for CR sub-resources. That is not the case right now, see discussion under [this issue](https://github.com/kubernetes/kubernetes/issues/72637).

In the absence of CR sub-resource support, the only way of implementing it is to switch to a custom aggregated API server. That means:
- Running api server deployment
- Moving instance CRD to be managed by this extension API server (breaking change)
- Running another controller-manager (? could we use the same one ?)
TODO: is this supported in kubectl somehow?

#### Pros

- More kubernetes native way of implementing such API with build in RBAC

#### Cons

- Lots of added complexity in terms of deployment and debugging
- Does not look like people are really doing this that much (they usually rather find a way how to make their use case work with plain CRDs)


### Introducing a new field in the Instance.Spec

- This is very similar to the sub-resource solution. Same pros and none of the API-server-related cons.

#### Pros

- Technically the simplest way of implementing the feature
- We can guard against the inconsistencies of plan triggering in the Instance webhook
- Immediate feedback for the CLI when e.g. a plan can not be started since webhook can decline requests with an status code and an error message

#### Cons
- Growing `Instance.Spec` and some denormalization of the data model 

### User Stories

- As an operator (person) I want to be able to run a custom plan on my KUDO operator in order to trigger non-default plan
- As a stateful service operator (person) I want to be able to run backup on my stateful service when a backup plan is defined.

### The Solution

After lots of thought experiments and prototyping around potential solutions, we decided to move on with introducing a new field to the `Instance.Spec`. The pros of keeping parameter updates, upgrades and directly triggered plans consistently and providing immediate user feedback far outweigh the data-model denormalization cons. Should Kubernetes implement support for custom sub-resources we might switch to that in the future.
