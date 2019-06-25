---
title: Concepts
type: docs
menu: docs
---

# Concepts

## Framework

High-level description of a deployable application. Contains metadata about the application (e.g., [Apache Kafka](https://github.com/kudobuilder/kudo/blob/master/config/samples/kafka-framework.yaml)).

## FrameworkVersion

Specific version of a deployable application, including configuration and lifecycle hooks for deployments, upgrades, and rollbacks (e.g., [Kafka version 2.4.0](https://github.com/kudobuilder/kudo/blob/master/config/samples/kafka-frameworkversion.yaml)).

## Instance

Resource created to manage an instance of specific FrameworkVersion. Instances are pets and have the same name throughout its entire lifecycle. (e.g., [Kafka 2.4.0 cluster with 1 broker](https://github.com/kudobuilder/kudo/blob/master/config/samples/kafka-instance.yaml)).

## Plan

Plans capture the individual steps of operational tasks such as a deploy, backup/restore, or version upgrade. Think of them as runbooks written in a structured way that can be executed by software. Plans are made up of phases, and phases have one or more steps.

## PlanExecution

Kudo-managed resource defining the inputs and status of an instance’s executable plans (e.g., upgrade kafka from version 2.4.0 -> 2.4.1).