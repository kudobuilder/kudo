---
kep-number: draft-20190307
title: Stable Kafka Example
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
      * [Goals](#goals)
      * [Non-Goals](#non-goals)

## Summary

In order to get KUDO ready for production it's useful to pick a specific workload, make it stable, and drive the KUDO roadmap from its requirements. Apache Kafka is a good candidate for the following reasons:

* it's one of the most popular stateful distributed systems
* its operator tasks are similar to many other stateful systems
* an example already exists in the KUDO repository

## Motivation

A team-sponsored production ready framework using KUDO helps with prioritizing features needed to achieve this goal.

### Goals

* Define criteria for what is considered a stable KUDO framework
* Identify features required for a stable Kafka and Zookeeper framework, create KEPs for them
* Create a stable Kafka framework
* Create a stable Zookeeper framework

### Non-Goals

* Any new KUDO features will not be covered by this KEP