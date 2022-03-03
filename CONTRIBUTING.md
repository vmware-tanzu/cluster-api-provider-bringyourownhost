# Contributing Guidelines

## As a Contributor
### Fork and branch
Local development has to be done in the forked repo of [cluster-api-provider-bringyourownhost](https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost). Here are some examples of meaningful branch names
* add-host-reservation-logic
* update-readme
* fix-byomachine-controller-flakes
* refactor-agent-unit-tests

### Writing tests
We expect our contributors to write unit / integration tests when making any code change. For tests like e2e, feel free to create a child issue and work on it separately.

#### Testing Framework
- We use [Ginkgo](https://onsi.github.io/ginkgo/) and [Gomega](https://onsi.github.io/gomega/) extensively for testing (unit / integration and e2e)
- For mocking interfaces and methods, we use the [Counterfeiter](https://github.com/maxbrunsfeld/counterfeiter) tool. Use this to generate fake implementations for your unit tests.

### Commit Message
If you are pairing on a PR, make sure to use [git-duet](https://github.com/git-duet/git-duet)

To learn more about how to write a good commit message, refer to this article - [How to write a Git commit message](https://chris.beams.io/posts/git-commit/)

At the minimum,
* the first line of the message should be concise and clear on the intent (this becomes the title of your PR)
* write (at least) another couple of lines explaining why the commit is necessary or what is the reasoning behind certain code logic

### Raising a PR
* all PRs should be raised against the main branch of [cluster-api-provider-bringyourownhost](https://github.com/vmware-tanzu/cluster-api-provider-bringyourownhost)
* all code changes should be accompanied with corresponding unit / integration tests (if for some reason, the code is not unit / integration testable, add enough justification in the PR. Although, this almost should never be the case.)

### Contributor License Agreement
All contributors to this project must have a signed Contributor License
Agreement (**"CLA"**) on file with us. The CLA grants us the permissions we
need to use and redistribute your contributions as part of the project; you or
your employer retain the copyright to your contribution. Before a PR can pass
all required checks, our CLA action will prompt you to accept the agreement.
Head over to [https://cla.vmware.com/](https://cla.vmware.com/) to see your
current agreement(s) on file or to sign a new one.

We generally only need you (or your employer) to sign our CLA once and once
signed, you should be able to submit contributions to any VMware project.

Note: if you would like to submit an "_obvious fix_" for something like a typo,
formatting issue or spelling mistake, you may not need to sign the CLA.

---

## As a Code Reviewer
We encourage code reviews by non-maintainers as well. If you are lacking context, please ask the author to better explain the change in the description.

Look for things like
* Is the code well-designed? Is it consistent with Cluster API contract?
* Are there ways to write simpler code? If so, please suggest
* Is the code well covered by unit / integration / e2e tests?
* Does the naming convention (variables / functions / types / methods) make sense?
* Are there enough comments? Note that comments should explain why rather than what