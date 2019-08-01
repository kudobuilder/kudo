---
title: Contributing
type: docs
menu: docs
---
# Contributing
The source code for both [KUDO](https://github.com/kudobuilder/kudo) and [KUDO Operators](https://github.com/kudobuilder/operators) lives on GitHub.  We welcome feature requests and bug reports in the form of [issues](https://help.github.com/en/articles/about-issues), and of course code - which includes documentation! - in the form of [pull requests](https://help.github.com/en/articles/about-pull-requests) (PRs).

There's a ton of stuff to do and there's opportunities to contribute in a variety of ways.  We'd suggest that newcomers look at issues tagged with ['good first issue'](https://github.com/kudobuilder/kudo/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22) and ['help wanted'](https://github.com/kudobuilder/kudo/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22) and then then jump into [#kudo on the Kubernetes Slack](https://kubernetes.slack.com/messages/kudo/) to discuss.

Please also take some time to read our [Contributing Guidelines](https://github.com/kudobuilder/kudo/blob/master/CONTRIBUTING.md).

## Raising an Issue
If you've hit a bug, have an idea for a new feature, or want to suggest some other kind of change then we welcome an issue detailing your problem or your suggestion.  Ideally we'd ask that people reach out in the Slack channel and join one of our weekly meetings so that other developers and users can help iterate.

## Creating a pull request
Yes please!  Bring us your code!  There's a whole lot of work to do, and we're committed to building an active community around KUDO in order to ensure its longevity.

PRs raised against either repo have a default template which help guide contributors to focus on the details necessary for a speedy review.  Again, please follow-up with discussion in the Slack channel.  We're also happy for people to submit draft PRs which can then be worked through with other members of the KUDO community.

## Reviewing a pull request
This process is adapted from the one defined for [contributing to Kubernetes](https://kubernetes.io/docs/contribute/intermediate/#review-a-pr) itself, so should be familiar.  We use [Prow](https://prow.k8s.io/) to help automate this process.

* Examine the PR description and read any associated issues or links for context;
* Look over all changed files, and if you have a comment or a question on any highlighted section then start a review;
* Continue to add comments using this review process and when you've finished, choose either 'comment' for general commentary or 'request changes' for anything you deem important enough to warrant further work;
* If you spot a relatively trivial error such as a typo or something that's not directly related to the stated purpose of the PR then you can let the submitter know by prefixing your review comment with `nit:`.  These are not necessarily blockers to the PR itself but it gives the author an opportunity to make amendments;
* If you think the PR is ready to be merged, then you can add the command `/approve` to your summary comment.  Note that only those listed in the approvers section of the [OWNERS](https://github.com/kudobuilder/kudo/blob/master/OWNERS) file can use this command;
* PRs can be assigned to an individual with the `/assign` command.  If you think a proposed change needs a specific person's input, use this command along with their GitHub username to get their attention;
* If a PR has the `lgtm` and / or the `approve` label then it will be merged automatically;
  * You can apply the `do-not-merge/hold` label in order to stop PRs from being merged automatically.

Typically, a PR needs a review and an approval from two [core developers](https://github.com/orgs/kudobuilder/people) in order to be merged. 
