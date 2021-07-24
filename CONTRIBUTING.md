# Contributing Guidelines

## As a Developer
### Fork and branch
Local development has to be done in the forked repo of [cluster-api-provider-byoh](https://github.com/vmware-tanzu/cluster-api-provider-byoh). Here are some examples of meaningful branch names
* add-host-reservation-logic
* update-readme
* fix-byomachine-controller-flakes
* refactor-agent-unit-tests

### Commit Message
If you are pairing on a PR, make sure to use [git-duet](https://github.com/git-duet/git-duet)

To learn more about how to write a good commit message, refer to this article - [How to write a Git commit message](https://chris.beams.io/posts/git-commit/)

At the minimum,
* the first line of the message should be concise and clear on the intent (this becomes the title of your PR)
* write (at least) another couple of lines explaining why the commit is necessary or what is the reasoning behind certain code logic
* should be authored and signed-off by username@vmware.com

### Raising a PR
Before you raise a PR, kindly run the test suite in your local environment and make sure everything is GREEN !!!
* all PRs should be raised against the main branch of [cluster-api-provider-byoh](https://github.com/vmware-tanzu/cluster-api-provider-byoh)
* folks can request reviews on the PR, but are also requested to post PRs in slack so that the entire team is aware and has a chance to review code

---
## As a Code Reviewer
All the members of the team are encouraged to review the code. If you are lacking context, please ask the author to better explain the change in the description or chat with them to understand the issue / feature.

Look for things like
* Is the code well designed? Is it consistent with Cluster API contract / rest of the providers ?
* Are there ways to write simpler code? If so, please suggest
* Is the code well covered by unit / integration / e2e tests?
* Does the naming convention (variables / functions / types / methods) make sense?
* Are there enough comments? Note that comments should explain the why rather than what