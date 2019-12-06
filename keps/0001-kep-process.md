---
kep-number: 1
title: KUDO Enhancement Proposal Process
short-desc: Description of the KEP-Process
authors:
  - "@fabianbaier"
owners:
  - "@fabianbaier"
  - "@gerred"
  - "@runyontr"
editor: "@fabianbaier"
creation-date: 2019-02-14
last-updated: 2019-02-19
status: implemented
---

# KUDO Enhancement Proposal Process

## Table of Contents

* [Kudo Enhancement Proposal Process](#kudo-enhancement-proposal-process)
  * [Metadata](#metadata)
  * [Table of Contents](#table-of-contents)
  * [Summary](#summary)
  * [Motivation](#motivation)
  * [Reference-level explanation](#reference-level-explanation)
      * [What type of work should be tracked by a KEP](#what-type-of-work-should-be-tracked-by-a-kep)
      * [KEP Template](#kep-template)
      * [KEP Metadata](#kep-metadata)
      * [KEP Workflow](#kep-workflow)
      * [Git and GitHub Implementation](#git-and-github-implementation)
      * [KEP Editor Role](#kep-editor-role)
      * [Important Metrics](#important-metrics)
      * [Prior Art](#prior-art)
  * [Graduation Criteria](#graduation-criteria)
  * [Drawbacks](#drawbacks)
  * [Alternatives](#alternatives)
  * [Unresolved Questions](#unresolved-questions)
  * [Mentors](#mentors)

## Summary

A standardized development process for KUDO is proposed in order to

- provide a common structure for proposing changes to KUDO
- ensure that the motivation for a change is clear
- allow for the enumeration stability milestones and stability graduation
  criteria
- persist project information in a Version Control System (VCS) for future
  KUDOnauts
- support the creation of _high value user facing_ information such as:
  - an overall project development roadmap
  - motivation for impactful user facing changes
- reserve GitHub issues for tracking work in flight rather than creating "umbrella"
  issues
- ensure community participants are successfully able to drive changes to
  completion across one or more releases while stakeholders are adequately
  represented throughout the process

This process is supported by a unit of work called a KUDO Enhancement Proposal or KEP.
A KEP attempts to combine aspects of a

- feature, and effort tracking document
- a product requirements document
- design document

into one file which is created incrementally in collaboration with one or more
owners.

## Motivation

A single GitHub Issue or Pull request seems to be required in order to understand
and communicate upcoming changes to KUDO. In a blog post describing the
[road to Go 2][], Russ Cox explains

> that it is difficult but essential to describe the significance of a problem
> in a way that someone working in a different environment can understand

as a project it is vital to be able to track the chain of custody for a proposed
enhancement from conception through implementation.

Without a standardized mechanism for describing important enhancements our
talented technical writers and product managers struggle to weave a coherent
narrative explaining why a particular release is important. Additionally for
critical infrastructure such as KUDO adopters need a forward looking road
map in order to plan their adoption strategy.

The purpose of the KEP process is to reduce the amount of "tribal knowledge" in
our community. By moving decisions from a smattering of mailing lists, video
calls and hallway conversations into a well tracked artifact this process aims
to enhance communication and discoverability.

A KEP is broken into sections which can be merged into source control
incrementally in order to support an iterative development process. An important
goal of the KEP process is ensuring that the process for submitting the content
contained in [design proposals][] is both clear and efficient. The KEP process
is intended to create high quality uniform design and implementation documents
for OWNERs to deliberate.

[road to Go 2]: https://blog.golang.org/toward-go2
[design proposals]: /contributors/design-proposals


## Reference-level explanation

### What type of work should be tracked by a KEP

The definition of what constitutes an "enhancement" is a foundational concern
for the KUDO project. Roughly any KUDO user or operator facing
enhancement should follow the KEP process: if an enhancement would be described
in either written or verbal communication to anyone besides the KEP author or
developer then consider creating a KEP.

Similarly, any technical effort (refactoring, major architectural change) that
will impact a large section of the development community should also be
communicated widely. The KEP process is suited for this even if it will have
zero impact on the typical user or operator.

As the local bodies of governance, OWNERs should have broad latitude in describing
what constitutes an enhancement which should be tracked through the KEP process.
OWNERs may find that helpful to enumerate what _does not_ require a KEP rather
than what does. OWNERs also have the freedom to customize the KEP template
according to their OWNER specific concerns. For example the KEP template used to
track FEATURE changes will likely have different subsections than the template for
proposing governance changes. However, as changes start impacting other aspects or
the larger developer community, the KEP process should be used
to coordinate and communicate.

Enhancements that have major impacts on multiple OWNERs should use the KEP process.
A single OWNER will own the KEP but it is expected that the set of approvers will span the impacted OWNERs.
The KEP process is the way that OWNERs can negotiate and communicate changes that cross boundaries.

KEPs will also be used to drive large changes that will cut across all parts of the project.
These KEPs will be owned among the OWNERs and should be seen as a way to communicate the most fundamental aspects of what Kubernetes is.

### KEP Template

The template for a KEP is precisely defined [here](0000-kep-template.md)

### KEP Metadata

There is a place in each KEP for a YAML document that has standard metadata.
This will be used to support tooling around filtering and display.  It is also
critical to clearly communicate the status of a KEP.

Metadata items:
* **kep-number** Required
  * Each proposal has a number.  This is to make all references to proposals as
    clear as possible.  This is especially important as we create a network
    cross references between proposals.
  * Before having the `Approved` status, the number for the KEP will be in the
    form of `draft-YYYYMMDD`.  The `YYYYMMDD` is replaced with the current date
    when first creating the KEP.  The goal is to enable fast parallel merges of
    pre-acceptance KEPs.
  * On acceptance a sequential dense number will be assigned.  This will be done
    by the editor and will be done in such a way as to minimize the chances of
    conflicts.  The final number for a KEP will have no prefix.
* **title** Required
  * The title of the KEP in plain language.  The title will also be used in the
    KEP filename.  See the template for instructions and details.
* **status** Required
  * The current state of the KEP.
  * Must be one of `provisional`, `implementable`, `implemented`, `deferred`, `rejected`, `withdrawn`, or `replaced`.
* **authors** Required
  * A list of authors for the KEP.
    This is simply the Github ID.
    In the future we may enhance this to include other types of identification.
* **owners** Required
  * An OWNER is the person or entity that works on the proposal.
  * OWNERs are listed as `@owner` where the name matches up with the Github ID.
  * The OWNER that is most closely associated with this KEP. If there is code or
    other artifacts that will result from this KEP, then it is expected that
    this OWNER will take responsibility for the bulk of those artifacts.
* **editor** Required
  * Someone to keep things moving forward.
  * If not yet chosen replace with `TBD`
  * Same name/contact scheme as `authors`
* **creation-date** Required
  * The date that the KEP was first submitted in a PR.
  * In the form `yyyy-mm-dd`
  * While this info will also be in source control, it is helpful to have the set of KEP files stand on their own.
* **last-updated** Optional
  * The date that the KEP was last changed significantly.
  * In the form `yyyy-mm-dd`
* **see-also** Optional
  * A list of other KEPs that are relevant to this KEP.
  * In the form `KEP-123`
* **replaces** Optional
  * A list of KEPs that this KEP replaces.  Those KEPs should list this KEP in
    their `superseded-by`.
  * In the form `KEP-123`
* **superseded-by**
  * A list of KEPs that supersede this KEP. Use of this should be paired with
    this KEP moving into the `Replaced` status.
  * In the form `KEP-123`


### KEP Workflow

A KEP has the following states

- `provisional`: The KEP has been proposed and is actively being defined.
  This is the starting state while the KEP is being fleshed out and actively defined and discussed.
  The OWNER has accepted that this is work that needs to be done.
- `implementable`: The approvers have approved this KEP for implementation and OWNERs create, if appropriate, 
  a [milestone](https://github.com/kudobuilder/kudo/milestones) to track implementation work.
- `implemented`: The KEP has been implemented and is no longer actively changed. OWNERs reflect
  the status change and close its matching milestone, if appropriate.
- `deferred`: The KEP is proposed but not actively being worked on.
- `rejected`: The approvers and authors have decided that this KEP is not moving forward.
  The KEP is kept around as a historical document.
- `withdrawn`: The KEP has been withdrawn by the authors.
- `replaced`: The KEP has been replaced by a new KEP.
  The `superseded-by` metadata value should point to the new KEP.

### Git and GitHub Implementation

KEPs are checked into under the `/keps` directory.
In the future, as needed we can add OWNER specific subdirectories.

New KEPs can be checked in with a file name in the form of `draft-YYYYMMDD-my-title.md`.
As significant work is done on the KEP the authors can assign a KEP number.
No other changes should be put in that PR so that it can be approved quickly and minimize merge conflicts.
The KEP number can also be done as part of the initial submission if the PR is likely to be uncontested and merged quickly.

### KEP Editor Role

Taking a cue from the [Python PEP process][], we define the role of a KEP editor.
The job of an KEP editor is likely very similar to the [PEP editor responsibilities][] and will hopefully provide another opportunity for people who do not write code daily to contribute to KUDO.

In keeping with the PEP editors which

> Read the PEP to check if it is ready: sound and complete. The ideas must make
> technical sense, even if they don't seem likely to be accepted.
> The title should accurately describe the content.
> Edit the PEP for language (spelling, grammar, sentence structure, etc.), markup
> (for reST PEPs), code style (examples should match PEP 8 & 7).

KEP editors should generally not pass judgement on a KEP beyond editorial corrections.
KEP editors can also help inform authors about the process and otherwise help things move smoothly.

[Python PEP process]: https://www.python.org/dev/peps/pep-0001/
[PEP editor responsibilities]: https://www.python.org/dev/peps/pep-0001/#pep-editor-responsibilities-workflow

### Important Metrics

It is proposed that the primary metrics which would signal the success or
failure of the KEP process are

- how many "enhancements" are tracked with a KEP
- distribution of time a KEP spends in each state
- KEP rejection rate
- PRs referencing a KEP merged per week
- number of issues open which reference a KEP
- number of contributors who authored a KEP
- number of contributors who authored a KEP for the first time
- number of orphaned KEPs
- number of retired KEPs
- number of superseded KEPs

### Prior Art

The KEP process as proposed was essentially stolen from the Kubernetes process that also is the [Rust RFC process][] which
itself seems to be very similar to the [Python PEP process][]

[Rust RFC process]: https://github.com/rust-lang/rfcs

## Drawbacks

Any additional process has the potential to engender resentment within the
community. There is also a risk that the KEP process as designed will not
sufficiently address the scaling challenges we face today. PR review bandwidth is
already at a premium and we may find that the KEP process introduces an unreasonable
bottleneck on our development velocity.

It certainly can be argued that the lack of a dedicated issue/defect tracker
beyond GitHub issues contributes to our challenges in managing a project like
KUDO, however, given that other large organizations, including GitHub
itself, make effective use of GitHub issues perhaps the argument is overblown.

The centrality of Git and GitHub within the KEP process also may place too high
a barrier to potential contributors, however, given that both Git and GitHub are
required to contribute code changes to KUDO today perhaps it would be reasonable
to invest in providing support to those unfamiliar with this tooling.

Expanding the proposal template beyond the single sentence description currently
required in the [features issue template][] may be a heavy burden for non native
English speakers and here the role of the KEP editor combined with kindness and
empathy will be crucial to making the process successful.

[features issue template]: https://www.github.com/kudobuilder/kudo/ISSUE_TEMPLATE.md

### GitHub issues vs. KEPs

The use of GitHub issues when proposing changes does not provide OWNERs good
facilities for signaling approval or rejection of a proposed change to KUDO
since anyone can open a GitHub issue at any time. Additionally managing a proposed
change across multiple releases is somewhat cumbersome as labels and milestones
need to be updated for every release that a change spans. These long lived GitHub
issues lead to an ever increasing number of issues open against
`kubernetes/features` which itself has become a management problem in the Kubernetes community.

In addition to the challenge of managing issues over time, searching for text
within an issue can be challenging. The flat hierarchy of issues can also make
navigation and categorization tricky. While not all community members might
not be comfortable using Git directly, it is imperative that as a community we
work to educate people on a standard set of tools so they can take their
experience to other projects they may decide to work on in the future. While
git is a fantastic version control system (VCS), it is not a project management
tool nor a cogent way of managing an architectural catalog or backlog; this
proposal is limited to motivating the creation of a standardized definition of
work in order to facilitate project management. This primitive for describing
a unit of work may also allow contributors to create their own personalized
view of the state of the project while relying on Git and GitHub for consistency
and durable storage.

## Unresolved Questions

- How reviewers and approvers are assigned to a KEP
- Example schedule, deadline, and time frame for each stage of a KEP
- Communication/notification mechanisms
- Review meetings and escalation procedure
