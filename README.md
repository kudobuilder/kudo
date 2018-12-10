# Maestro

Maestro is a Kubernetes operator built for deploying software with interesting lifecycles.

## Motivation

- Inspired by DC/OS Commons
- Need for an abstraction framework around complex software lifecycles
- ???

## Installation Instructions

- Clone
- `make install` to install CRDs
- `make run` to run the Operator

## Design Philosophy

- Centered around CRDs for defining frameworks, versions, and instances
- Instances execute series of plans
- Plans, phases, steps, etc.

## Packaging Format

- Framework CRD
- FrameworkVersion CRD
- Instance CRD

## Plans and PlanExecution

- TODO

### Code of Conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

[owners]: https://git.k8s.io/community/contributors/guide/owners.md
[creative commons 4.0]: https://git.k8s.io/website/LICENSE

