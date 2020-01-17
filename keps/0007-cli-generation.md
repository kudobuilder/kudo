---
kep-number: 7
title: CLI Skeleton generator
short-desc: Generate operator skeletons for operator developers
authors:
  - "@djannot"
owners:
  - "@djannot"
  - "@fabianbaier"
editor: "@fabianbaier"
creation-date: 2019-04-18
status: provisional
---

# Skeleton generator

## Table of Contents

  * [Table of Contents](#table-of-contents)
  * [Summary](#summary)
  * [Motivation](#motivation)

Created by [gh-md-toc](https://github.com/ekalinin/github-markdown-toc)

## Summary

In order to develop a new operator using KUDO, you always need to create several `Plans` (`deploy`, `update`, `upgrade`, ...) and for each plan to create several `Phases`, `Steps` and `Tasks`.

Also, any Operator will have a `Task` in the `deploy` plan to create similar Kubernetes objects (`StatefulSets`, `Services`, ...).

A Skeleton generator could create `yaml` templates with the common `Plans`, `Phases`, `Steps` and `Tasks`.

This generator should be part of the KUDO CLI.

With the new proposed approach based on dynamic CRDs, these elements will be defined in separated `yaml` files, so it will be even easier to create a Skeleton generator with a clear directory structure.

## Motivation

Having a Skeleton generator would help people getting started faster when they want to develop a new KUDO operator.

It would also make the operator created using this generator more readable as they would follow a naming convention for the different objects (`StatefulSets`, `Services`, ...).
