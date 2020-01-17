---
kep-number: 3
title: KEP CLI Proposal
short-desc: Initial CLI description
authors:
  - "@fabianbaier"
owners:
  - "@fabianbaier"
editor: TBD
creation-date: 2019-02-14
last-updated: 2019-02-20
status: implementable
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
        * [Installing an Operator](#installing-an-operator)
        * [Uninstalling an Operator](#uninstalling-an-operator)
        * [Listing running Instances](#listing-running-instances)
        * [Getting the status of an Instance](#getting-the-status-of-an-instance)
        * [Start specific plans](#start-specific-plans)
        * [Get the history of Planexecutions](#get-the-history-of-planexecutions)
        * [Executing into a particular Instance](#executing-into-a-particular-instance)
        * [Run Operator specific commands](#run-operator-specific-commands)
        * [Shell into containers of an Operator](#shell-into-containers-of-an-operator)
        * [Read all combined logs of an Operator](#read-all-combined-logs-of-an-operator)
     * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
     * [Risks and Mitigations](#risks-and-mitigations)
  * [Graduation Criteria](#graduation-criteria)
  * [Implementation History](#implementation-history)

## Summary

This KEP aims to improve the end user experience via a CLI that makes the life of an operator easier. Right now,
interacting with KUDO consists of a chain of cumbersome steps that require specific business logic. By providing one
global entry point for the user (e.g. to install or interact with operators, get the status of current plans etc.) we
can abstract complexity from the user and drive adaptation.

## Motivation

Currently, installing your `Operators` requires not just the proper configuration of your YAML files but also multiple 
interactions with `kubectl`. Having CLI tooling to simplify the path for cluster operators to deployment and maintenance 
 of installed `Operators` adds not just value across the community but also helps implementing best practices when using KUDO.

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

#### Installing an Operator

As a user I want to be able to install an operator to my cluster.

#### Uninstalling an Operator

As a user I want to be able to uninstall an operator from my cluster.

#### Listing running Instances

As a user I want to be able to list current installed operators in my namespace.

#### Getting the status of an Instance

As a user I want to be able to get the status of all plans of an Instance.

#### Start specific plans

As a user I want to be able to trigger specific plans of an Instance.

#### Get the history of Planexecutions

As a user I want to be able to audit the trail of executed plans.

#### Executing into a particular Instance

As a user I want to be able to execute into my Instance environment.

#### Run Operator specific commands

As an application operator, I want to be able to run CLI tooling, built for the Operator against an Instance.

#### Shell into containers of an Operator

As a user I want to be able to get a shell to a specific operator.

#### Read all combined logs of an Operator

As a user I want to be able to read logs to a specific operator and its components.

### Implementation Details/Notes/Constraints

Some caveats:

- Airgapped clusters
- Application logic with extra plugins for specific operators
- To be continued...

### Risks and Mitigations

TBD

## Graduation Criteria

If we see adoption in the community.

## Implementation History

TBD