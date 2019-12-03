---
kep-number: 6
title: Stable Kafka Example
short-description: Description for a full Kafka Operator
authors:
  - "@guenter"
owners:
  - TBD
  - "@guenter"
editor: TBD
creation-date: 2019-03-07
status: provisional
---

# Stable Kafka Example

## Table of Contents

* [Stable Kafka Example](#stable-kafka-example)
  * [Table of Contents](#table-of-contents)
  * [Summary](#summary)
  * [Motivation](#motivation)
  * [Proposal](#proposal)
      * [Goals](#goals)
      * [Non-Goals](#non-goals)
      * [User Stories](#user-stories)
        * [Install](#install)
        * [Cluster Size](#cluster-size)
        * [Multiple Clusters](#multiple-clusters)
        * [ZooKeeper Dependency](#zookeeper-dependency)
        * [Readiness Check](#readiness-check)
        * [Health Check](#health-check)
        * [Vertical Scaling](#vertical-scaling)
        * [Horizontal Scaling](#horizontal-scaling)
        * [Configuration Changes](#configuration-changes)
        * [Handle Failures](#handle-failures)
        * [Upgrades](#upgrades)
        * [TLS](#tls)
        * [Kerberos](#kerberos)
        * [Secrets Integration](#secrets-integration)
        * [Logging](#logging)
        * [JMX Metrics](#jmx-metrics)

Created by [gh-md-toc](https://github.com/ekalinin/github-markdown-toc)

## Summary

In order to get KUDO ready for production it's useful to pick a specific workload, make it stable, and drive the KUDO roadmap from its requirements. Apache Kafka is a good candidate for the following reasons:

* it's one of the most popular stateful distributed systems
* its operator tasks are similar to many other stateful systems
* an example already exists in the KUDO repository

## Motivation

A team-sponsored production ready operator using KUDO helps with prioritizing features needed to achieve this goal.

## Proposal

### Goals

* Define criteria for what is considered a stable KUDO operator
* Identify features required for a stable Kafka and ZooKeeper operator, create KEPs for them
* Create a stable Kafka operator
* Create a stable ZooKeeper operator

### Non-Goals

* Any new KUDO features will not be covered by this KEP

### User Stories

#### Install

As a Kafka administrator I want to deploy a Kafka cluster with a single command so I can get started quickly.

#### Cluster Size

As a Kafka administrator I want to deploy Kafka clusters of different sizes so I can size my cluster to the application.

#### Multiple Clusters

As a Kafka administrator I want to deploy multiple Kafka clusters on the same Kubernetes cluster so that I can use my resources efficiently.

#### ZooKeeper Dependency

As a Kafka administrator I want to deploy Kafka with a dedicated ZooKeeper, so that I can minimize the impact of ZooKeeper downtime to a single Kafka cluster and not learn how to manage it.

#### Readiness Check

As a Kafka administrator I want to have an indication of when my cluster is ready so that know when to start using it.

#### Health Check

As a Kafka administrator I want to have an indication of cluster health so that know when it is degraded.

#### Vertical Scaling

As a Kafka administrator I want to change the resources used by Kafka brokers (CPU, memory) with minimal disruption to clients so I can scale my cluster vertically when the workload increases.

#### Horizontal Scaling

As a Kafka administrator I want to change the number of Kafka brokers in a cluster with minimal disruption to clients so I can scale my cluster horizontally when the workload increases.

#### Configuration Changes

As a Kafka administrator I want to deploy configuration changes to the cluster with minimal disruption to clients so I can make changes without creating a new cluster.

#### Handle Failures

When a Pod corresponding to Kafka Broker is restarted (because of a Kubelet failure, for example), I want a task to run to verify the health of the Broker and to perform a maintenance if needed

#### Upgrades

As a Kafka administrator I want to upgrade to a new Kafka version with minimal disruption to clients so I can deploy bug fixes and enable new features.

#### TLS

As a Kafka administrator I want to optionally deploy Kafka with TLS enabled so that I can secure it.

#### Kerberos

As a Kafka administrator I want to optionally deploy Kafka with Kerberos enabled so that I can secure it.

#### Secrets Integration

As a Kafka administrator I want to use a secrets store for secure storage and distribution of the credentials for Kafka.

#### Logging

As a Kafka administrator I want to have easy access to all relevant logs so I can easily debug problems.

#### JMX Metrics

As a Kafka administrator I want to have easy access to Kafka's JMX metrics so I can easily debug problems.
