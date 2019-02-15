---
kep-number: draft-20190214
title: KEP CLI Proposal
authors:
  - "@fabianbaier"
owners:
  - TBD
  - "@fabianbaier"
editor: TBD
creation-date: 2019-02-14
status: provisional
---

# KEP CLI Proposal

## Table of Contents

 * [KEP CLI Proposal](#kep-cli-proposal)
      * [Table of Contents](#table-of-contents)
      * [Summary](#summary)
      * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals)
      * [Proposal](#proposal)
         * [User Stories](#user-stories)
            * [Installing a framework](#installing-a-framework)
            * [Uninstalling a framework](#uninstalling-a-framework)
            * [Listing running frameworks](#listing-running-frameworks)
            * [Getting the status of a running framework](#getting-the-status-of-a-running-framework)
            * [Start specific plans](#start-specific-plans)
            * [Get the history of planexecutions](#get-the-history-of-planexecutions)
            * [Executing into a particular framework](#executing-into-a-particular-framework)
            * [Run framework specific commands](#run-framework-specific-commands)
            * [Shell into containers of a framework](#shell-into-containers-of-a-framework)
            * [Read all combined logs of a framework](#read-all-combined-logs-of-a-framework)
         * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
         * [Risks and Mitigations](#risks-and-mitigations)
      * [Graduation Criteria](#graduation-criteria)
      * [Implementation History](#implementation-history)

## Summary

This KEP aims to improve the end user experience via a CLI that makes the life of an operator easier. Right now,
interacting with KUDO consists of a chain of cumbersome steps that require specific business logic. By providing one
global entry point for the user (e.g. to install or interact with frameworks, get the status of current plans etc.) we
can abstract complexity from the user and drive adaptation.

## Motivation

Currently, installing your `Frameworks` requires not just the proper configuration of your YAML files but also multiple 
interactions with `kubectl`. Having CLI tooling to simplify the path for cluster operators to deployment and maintenance 
 of installed `Frameworks` adds not just value across the community but also helps implementing best practices when using KUDO.

### Goals

The goal of this KEP is to have a CLI binary that can be used under the Kubectl Plugin System to work with KUDO.

- We should drive making the adaption to KUDO easier
- Not to confuse flags
- Providing metrics for usage and error reporting

### Non-Goals

Non-Goals are:

- Just building another wrapper around `kubectl`
- Adding complexity

## Proposal

This is where we get down to the nitty gritty of what the proposal actually is.

### User Stories

#### Installing a framework

As a user I want to be able to install a framework to my cluster.

#### Uninstalling a framework

As a user I want to be able to uninstall a framework from my cluster.

#### Listing running frameworks

As a user I want to be able to list current installed frameworks in my namespace.

#### Getting the status of a running framework

As a user I want to be able to get the status of plans a framework can have.

#### Start specific plans

As a user I want to be able to trigger specific plans of a framework.

#### Get the history of planexecutions

As a user I want to be able to audit the trail of executed plans.

#### Executing into a particular framework

As a user I want to be able to execute into my framework environment.

#### Run framework specific commands

As a user I want to be able to use framework specific logic, e.g. in Kafka create a topic or in Flink to upload a new job.

#### Shell into containers of a framework

As a user I want to be able to get a shell to a specific framework.

#### Read all combined logs of a framework

As a user I want to be able to read logs to a specific framework and its components.

### Implementation Details/Notes/Constraints

Some caveats:

- Airgapped clusters
- Application logic with extra plugins for specific frameworks
- To be continued...

### Risks and Mitigations

TBD

## Graduation Criteria

If we found adaption in the community.

## Implementation History

TBD