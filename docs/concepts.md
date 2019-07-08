---
title: Concepts
type: docs
menu: docs
---

# Concepts

## Application

Any kind of software that you would want to run in your Kubernetes cluster. It can be as simple as cleanup job or as complex as distributed system like Kafka.

## Operator

High-level description of a deployable application to be run in a k8s cluster. Contains metadata about the application (e.g., [Apache Kafka](https://github.com/kudobuilder/kudo/blob/master/config/samples/kafka-operator.yaml)).
You can have multiple versions of Kafka ready to be installed in your cluster, all will belong to the same Operator.

## OperatorVersion

Specific version of a deployable application, including configuration and lifecycle hooks for deployments, upgrades, and rollbacks (e.g., [Kafka version 2.4.0](https://github.com/kudobuilder/kudo/blob/master/config/samples/kafka-operatorversion.yaml)).
This is already complete definition of application to be installed (except overridable parameters). By adding OperatorVersion to your cluster, no application is running yet.

## Instance

When you create an instance, you provide missing parameters for the installed OperatorVersion. Creating an instance typically causes rendering of those parameters in your templates, such as services, pods or statefulsets. Once rendered these objects will then be applied with the given parameters to your cluster.
Instances have the same name throughout its entire lifecycle. (e.g., [Kafka 2.4.0 cluster with 1 broker](https://github.com/kudobuilder/kudo/blob/master/config/samples/kafka-instance.yaml)).

You can create multiple instance of an OperatorVersion in your cluster (e.g. different Kafka instances for different teams).

## Plan

Operator typically define several plans. Plans capture the individual steps of operational tasks. Think of them as runbooks written in a structured way that can be executed by software. Plans are made up of phases, and phases have one or more steps.

Every OperatorVersion must contain a `deploy` plan which is the default plan to deploy an application to the cluster. For more complex systems, you would want to define a plan for backup and restore or upgrade.

## PlanExecution

Every time a plan is executed, the corresponding PlanExecution CRD is stored with inputs and status of the plan (e.g., when you upgrade Kafka from version 2.4.0 -> 2.4.1).
You can query the status of any plan via the CLI.
