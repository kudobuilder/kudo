---
kep-number: 11
title: Diagnostic Template
authors:
  - "@gerred"
  - "@meichstedt"
owners:
  - "@gerred"
  - "@meichstedt"
creation-date: 2019-04-11
last-updated: 2019-04-11
status: provisional
---

# KUDO Diagnostic Template

- [KUDO Diagnostic Template](#kudo-diagnostic-template)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)

## Summary

The KUDO Diagnostic Template is intended to provide an opinionated template for issues when using frameworks for KUDO and creating frameworks with KUDO. This is intended to supersede `ISSUE_TEMPLATE.md` and provide an opinionated set of diagnostic information, logs, and tooling required to adequately diagnose and respond to issues with KUDO.

## Motivation

Engineering teams have many incoming priorities on top of work defined upfront during sprint planning. All time spent triaging issues takes away from estimating, prioritizing, and fixing issues. This KEP provides a framework for quickly triaging and identifying issues within KUDO.

### Goals

- Define criteria for submitting KUDO issues
- Define tooling, logs, and metrics needed to support, diagnose, and fix issues reported

### Non-Goals

- Define frameworks for supporting, diagnosing, and fixing frameworks built using KUDO

## Proposal

Current template requires the following information:

1. What happened
2. What was expected
3. Steps to reproduce
4. Environment
   a. Kubernetes version (use kubectl version):
   b. Kudo version (use kubectl kudoctl version):
   c. Framework:
   d. Frameworkversion:
   e. Cloud provider or hardware configuration:
   f. OS (e.g. from /etc/os-release):
   g. Kernel (e.g. uname -a):
   h. Install tools:
   i. Others:
