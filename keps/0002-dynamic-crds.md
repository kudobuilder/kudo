---
kep-number: 2
title: Dynamic CRDs
authors:
  - "@gerred"
owners:
  - TBD
editor: TBD
creation-date: 2019-02-14
last-updated: 2019-02-14
status: provisional
---

# Framework-specific Dynamic Custom Resource Definitions

## Table of Contents

A table of contents is helpful for quickly jumping to sections of a KEP and for highlighting any additional information provided beyond the standard KEP template.
[Tools for generating][] a table of contents from markdown are available.

- [Framework-specific Dynamic Custom Resource Definitions](#framework-specific-dynamic-custom-resource-definitions)
  - [Table of Contents](#table-of-contents)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [User Stories [optional]](#user-stories-optional)
      - [Story 1](#story-1)
      - [Story 2](#story-2)
    - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Graduation Criteria](#graduation-criteria)
  - [Implementation History](#implementation-history)
  - [Drawbacks [optional]](#drawbacks-optional)
  - [Alternatives [optional]](#alternatives-optional)
  - [Infrastructure Needed [optional]](#infrastructure-needed-optional)

[tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Summary

This KEP aims to make the end user experience for frameworks more specific to the business domain they represent. By implementing dynamic CRDs, frameworks will be able to represent their components in a declarative way, specific to their framework. Additionally, it enables framework developers to provide day 2 operations tasks as CRDs, complete with their own plans and tasks.

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

Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of the system.
The goal here is to make this feel real for users without getting bogged down.

#### Story 1

#### Story 2

### Implementation Details/Notes/Constraints [optional]

What are the caveats to the implementation?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they releate.

### Risks and Mitigations

What are the risks of this proposal and how do we mitigate.
Think broadly.
For example, consider both security and how this will impact the larger kubernetes ecosystem.

## Graduation Criteria

How will we know that this has succeeded?
Gathering user feedback is crucial for building high quality experiences and owners have the important responsibility of setting milestones for stability and completeness.
Hopefully the content previously contained in [umbrella issues][] will be tracked in the `Graduation Criteria` section.

[umbrella issues]: https://github.com/kubernetes/kubernetes/issues/42752

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling owner acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first KUDO release where an initial version of the KEP was available
- the version of KUDO where the KEP graduated to general availability
- when the KEP was retired or superseded

## Drawbacks [optional]

Why should this KEP _not_ be implemented.

## Alternatives [optional]

Similar to the `Drawbacks` section the `Alternatives` section is used to highlight and record other possible approaches to delivering the value proposed by a KEP.

## Infrastructure Needed [optional]

Use this section if you need things from the project/owner.
Examples include a new subproject, repos requested, github details.
Listing these here allows a owner to get the process for these resources started right away.
