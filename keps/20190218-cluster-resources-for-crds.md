---
kep-number: 0
title: My First KEP
authors:
  - "@runyontr"
owners:
  - "@runyontr"
editor: TBD
creation-date: 2019-02-18
last-updated: 2019-02-18
status: provisional
---

# cluster-resources-for-crds

In order to enable the creation use of certain CRDs, some cluster scoped objects may need to be created.  This should not be
part of the creation of a particular CRD instantiation since the deletion of that instance would remove the 
dependency from all objects.  Allowing a Framework or FrameworkVersion to define a set of Cluster objects that are present
to support the creation and management of CRDs would circumvent the CRD from having to create and manage the object.

To get started with this template:
1. **Make a copy of this template.**
  Name it `YYYYMMDD-my-title.md`.
1. **Fill out the "overview" sections.**
  This includes the Summary and Motivation sections.
1. **Create a PR.**
  Assign it to owner(s) that are sponsoring this process.
1. **Merge early.**
  Avoid getting hung up on specific details and instead aim to get the goal of the KEP merged quickly.
  The best way to do this is to just start with the "Overview" sections and fill out details incrementally in follow on PRs.
  View anything marked as a `provisional` as a working document and subject to change.
  Aim for single topic PRs to keep discussions focused.
  If you disagree with what is already in a document, open a new PR with suggested changes.

The canonical place for the latest set of instructions (and the likely source of this file) is [here](/keps/0000-kep-template.md).

The `Metadata` section above is intended to support the creation of tooling around the KEP process.
This will be a YAML section that is fenced as a code block.
See the [KEP process](/keps/0001-kep-process.md) for details on each of these items.

## Table of Contents

A table of contents is helpful for quickly jumping to sections of a KEP and for highlighting any additional information provided beyond the standard KEP template.
[Tools for generating][] a table of contents from markdown are available.

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [User Stories [optional]](#user-stories-optional)
      * [Story 1](#story-1)
      * [Story 2](#story-2)
    * [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)
* [Drawbacks [optional]](#drawbacks-optional)
* [Alternatives [optional]](#alternatives-optional)
* [Infrastructure Needed [optional]](#infrastructure-needed-optional)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Summary

The `Summary` section is incredibly important for producing high quality user focused documentation such as release notes or a development road map.
It should be possible to collect this information before implementation begins in order to avoid requiring implementors to split their attention between writing release notes and implementing the feature itself.
KEP editors should help to ensure that the tone and content of the `Summary` section is useful for a wide audience.

A good summary is probably at least a paragraph in length.

## Motivation

This section is for explicitly listing the motivation, goals and non-goals of this KEP.
Describe why the change is important and the benefits to users.
The motivation section can optionally provide links to [experience reports][] to demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports

### Goals

List the specific goals of the KEP.
How will we know that this has succeeded?

### Non-Goals

What is out of scope for his KEP?
Listing non-goals helps to focus discussion and make progress.

## Proposal

This is where we get down to the nitty gritty of what the proposal actually is.

### User Stories [optional]


#### Story 1

A database Framework may want a Restic server deployed as part of the Framework to provide a central location for storing backups. 
The capability here would define a namespace, service and deployment that provides Restic to the instance of the Framework to use
by default.


#### Story 2

A Framework could leverage a MutatingWebHook for modifying pods deployments based on Node metadata (e.g. use a different image for nodes configured with GPU, PMEM, etc)


#### Story 3

The creation of a CRD should be controlled by a ClusterRole that defines permissions on who can create instances of the CRD.  These
should probably be created

#### Story 4

FrameworkVersions require the existance of CRDs that are not controlled by Kudo (e.g. ETCD Operator) and require those to be installed when FV is enabled.

### Implementation Details/Notes/Constraints [optional]



### Risks and Mitigations



## Graduation Criteria

The MySQL Framework could be modified to deploy a central repo for backups and leverage those in each instance of MySQL

## Implementation History

* Initial KEP - 20190218

## Drawbacks [optional]

* More complicated Framework/FrameworkVersion specs
* Implications of FrameworkVersion installation making MORE cluster level changes than just a CRD.
