# Contributing Guidelines

## Sign the CLA

Kubernetes projects require that you sign a Contributor License Agreement (CLA) before we can accept your pull requests.

Please see [https://git.k8s.io/community/CLA.md](https://git.k8s.io/community/CLA.md) for more info.

## Contributing Steps

1. Submit an issue describing your proposed change to the repo in question.
2. The [repo owners](https://github.com/kudobuilder/kudo/blob/master/OWNERS) will respond to your issue promptly.
3. If your proposed change is accepted, and you haven't already done so, sign a Contributor License Agreement (see details above).
4. Fork the desired repo, develop and test your code changes.
5. Submit a pull request.
6.
## How to build Kudo locally

TBD

## Community, discussion, and support

Learn how to engage with the Kubernetes community on the [community page](community page).

You can reach the maintainers of this project at:

[Slack](https://kubernetes.slack.com/messages/kudo/)
[Mailing List](https://groups.google.com/forum/#!forum/kubernetes-kudo)

## Is My Thing an Enhancement?

We are trying to figure out the exact shape of an enhancement. Until then here are a few rough heuristics.

An enhancement is anything that:

- a blog post would be written about after its release (ex. [minikube](https://kubernetes.io/blog/2016/07/minikube-easily-run-kubernetes-locally/), [StatefulSets](https://kubernetes.io/blog/2016/07/thousand-instances-of-cassandra-using-kubernetes-pet-set/), [rkt container runtime](https://kubernetes.io/blog/2016/07/rktnetes-brings-rkt-container-engine-to-kubernetes/))
- requires multiple parties/owners participating to complete
- will be graduating from one stage to another (ex. alpha to beta, beta to GA)
- needs significant effort or changes KUDO in a significant way (ex. something that would take 10 person-weeks to implement, introduce or redesign a system component)
- impacts the UX or operation of KUDO substantially such that engineers using KUDO will need retraining
- users will notice and come to rely on

It is unlikely an enhancement if it is:
- fixing a flaky test
- refactoring code
- performance improvements, which are only visible to users as faster API operations, or faster control loops
- adding error messages or events

If you are not sure, ask someone in the [#kudo](https://kubernetes.slack.com/messages/kudo/) channel on Slack or ping someone listed in [OWNERS](https://github.com/kudobuilder/kudo/blob/master/OWNERS).

### When to Create a New Enhancement Issue

Create an issue in this repo once you:
- have circulated your idea to see if there is interest
   - through Community Meetings, KUDO meetings, KUDO mailing lists, or an issue in github.com/kudobuilder/kudo
- (optionally) have done a prototype in your own fork
- have identified people who agree to work on the enhancement
  - many enhancements will take several releases to progress through Alpha, Beta, and Stable stages
  - you and your team should be prepared to work on the approx. 9mo - 1 year that it takes to progress to Stable status
- are ready to be the project-manager for the enhancement

## Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](https://github.com/kudobuilder/kudo/blob/master/code-of-conduct.md).
