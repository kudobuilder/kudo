---
kep-number: draft-20200225
title: Reading parameter values from a file
short-desc: Reading parameter values from a file
authors:
  - "@porridge"
owners:
  - "@porridge"
editor: @porridge
creation-date: 2020-02-25
last-updated: 2020-02-25
status: provisional
see-also:
  - KEP-24
  - KEP-25
---

# Reading parameter values from a file

## Table of Contents

* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [User Stories](#user-stories)
    * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
* [Implementation History](#implementation-history)

## Summary

This KEP describes an alternative way to provide parameter values when installing or
updating an operator instance.

## Motivation

This has two motivations:
1. A better way for a user to store parameter values than a shell command line.
1. A more convenient way to pass complex parameter values, especially in the light of
coming structured parameter values (see [KEP-24](https://github.com/kudobuilder/kudo/pull/1356)).

See [User Stories](#user-stories) below for more details.

### Goals

- Make it possible to store parameter values in a file, and pass them in this form
to any `kubectl kudo` command which currently takes the `-p` / `--parameter` flag.
- Support both currently supported (i.e. `string`) and planned (see [KEP-24](https://github.com/kudobuilder/kudo/pull/1356))
types of parameter values.

### Non-Goals

- Make it possible for a user to pass any other command line flags through the file.
- Provide any templating or other pre-processing capabilities for the parameter value file.
 - Providing any new way of storage on the back-end.  This is strictly a way of passing parameters to the CLI.
## Proposal

The idea is to introduce a new command-line flag: `--parameter-file`, in every place that
currently accepts the `--parameter` flag.
This flag should require a single argument, interpreted as a path to a YAML file.
This file should contain parameters a mapping of strings to whatever values are supported as parameter values.

### User Stories

Currently the only mechanism for passing values to parameters is via the `--parameter`
command line flag (or its short form `-p`).

This means that the only way to keep track of what parameters were overridden with what
values is to store the full command line to `kubectl kudo` (or some other form from which
a command line can be generated with a custom tool).

This presents the following two basic problems.

#### Storing parameter values in version control

Some operators
([kafka](https://github.com/kudobuilder/operators/blob/master/repository/kafka/operator/params.yaml),
[cassandra](https://github.com/kudobuilder/operators/blob/master/repository/cassandra/3.11/operator/operator.yaml))
declare nearly 200 parameters which can be overridden, and we can expect this number to grow.

While it is unlikely that a given operator user will override every single one of these parameters,
passing them on the command line in large numbers gets inconvenient.
For example it is hard to provide comments for individual parameter values.

Being able to keep parameter values as a YAML file which can be directly read by
`kubectl kudo` is a good fit for storing configuration in version control.

#### Passing complex and/or long parameter values

Passing long parameter values through shell command-line, with the need to escape
shell meta-characters or newlines may become a pain.

These problems will be amplified once KUDO supports parameter value types other than
plain strings, such as nested dictionaries, etc.

Keeping values in a YAML file means they can be viewed and edited with tools more
powerful than shell command line editing, such as a text editor with syntax highlighting.

### Implementation Details/Notes/Constraints

Currently it is possible to provide the same parameter multiple times with
different values on the same command line. In this case the last value wins.

Similarly, it should be possible on a single command line to pass a value to
a parameter simultaneously via a `--parameter` flag and inside a file passed via
a `--parameter-file` flag.

It should also be possible to pass multiple `--parameter-file` flags with different
file paths. CLI should then load and merge all of them.

We need to document precedence rules in this case. Depending on the capabilities
of command-line parsing library we use (in the future) it might be safest to
say that `--parameter-file` flags are processed first, and `--parameter` flags later.
This is because the command-line parsing library might not allow us to discover
the relative order of different flags.

## Implementation History

- 2020-02-25 - Initial draft. (@porridge)
