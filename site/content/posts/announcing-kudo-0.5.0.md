---
title: Announcing KUDO 0.5.0
date: 2019-08-05
---

We are proud to announce the release of KUDO v0.5.0! This release enhances the test harness and KUDO CLI making it easier to perform updates and upgrades on your operators.

## Release Highlights

### kudo upgrade command
With new CLI installed you can now use `kubectl kudo upgrade` command to upgrade running instance on your cluster from one version of operator to another one. That means you can now automatically update for example from one version of Kafka to another. Run `kubectl kudo upgrade --help` for more details.

### kudo update command
`kubectl kudo update` command provides a new way to patch parameters on a running instance of an operator. So for example if Kafka exposes parameter `BROKER_MEM_LIMIT` you can run `kudo update --instance kafka -p "BROKER_MEM_LIMIT=4444Mi"` to change it for already installed Kafka. Run `kubectl kudo update --help` for more details.

### Ability to execute kubectl commands using test harness
If you're using test harness to write your tests (and you should!) you can now execute kubectl commands as part of your steps. See [this test](https://github.com/kudobuilder/kudo/tree/master/test/integration/cli-install) for a reference.

## What's Next?

Next minor planned release will be 0.6.0. We will focus on implementing extensions as described in [KEP-12](https://github.com/kudobuilder/kudo/blob/master/keps/0012-operator-extensions.md) as well as improvements for observability and debugging.
See the [KUDO Roadmap](https://github.com/orgs/kudobuilder/projects/2) for details.

[Get started](/docs/getting-started) with KUDO today. Our [community](/docs/community) is ready for feedback to make KUDO even better!
