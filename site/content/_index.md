---
title: Kubernetes Universal Declarative Operator (KUDO)
type: docs
---

## What is KUDO?

[Kubernetes Universal Declarative Operator (KUDO)](https://github.com/kudobuilder/kudo) provides a declarative approach to building production-grade Kubernetes Operators covering the entire application lifecycle. An operator is a way to package and manage a Kubernetes application using Kubernetes APIs. Building an Operator usually involves implementing a [custom resource and controller](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/), which typically requires thousands of lines of code and a deep understanding of Kubernetes. KUDO instead provides a universal controller that can be configured via a declarative spec (YAML files) to operate any workload.

KUDO-based Operators don't require any code in most cases, which significantly accelerates the development of Operators. It also eliminates sources of error and code duplication.

## When should I use KUDO?

When you need more than just `kubectl apply -f` to run your application.
When you want to provide application lifecycle scripts to application operators, but don’t want to provide scripts/documentation for them to follow.
When you don’t want to write your own operator.

## Where can I learn more about KUDO?

 Please take a look at the [github repo](https://github.com/kudobuilder/kudo) and connect to the community using [Slack](https://kubernetes.slack.com/messages/kudo/) or [Mailing List](https://groups.google.com/d/forum/kudobuilder).
