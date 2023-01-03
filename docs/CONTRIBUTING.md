<!--
Copyright © 2022-2023 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
   
    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->
# How to Contribute

Become one of the contributors to this project! We thrive to build a welcoming and open
community for anyone who wants to use the project or contribute to it. There are
just a few small guidelines you need to follow. To help us create a safe and positive community experience for all, we require all participants to adhere to the [Code of Conduct](CODE_OF_CONDUCT.md). 

# Table of Content:

* [Become a contributor](#Become-a-contributor)
* [Submitting issues](#Submitting-issues)
* [Triage issues](#Triage-issues)
* [Your first contribution](#Your-first-contribution)
* [Branching](#Branching)
* [Signing your commits](#Signing-your-commits)
* [Pull requests](#Pull-requests)
* [Code reviews](#Code-reviews)
* [TODOs in the code](#TODOs-in-the-code)

## Become a contributor

You can contribute to this project in several ways. Here are some examples:

* Contribute to the documentation and codebase.
* Report and triage bugs.
* Create feature requests
* Fix bugs and implement features
* Write technical documentation and blog posts, for users and contributors.
* Help others by answering questions about this project.

## Submitting issues

All issues related to CERT-CSI should be submitted [here](<LINK_TO_GITHUB_ISSUES>). Issues will be triaged and labels will be used to indicate the type of issue. This section outlines the types of issues that can be submitted.  

### Report bugs

Before submitting a new issue, make sure someone hasn't already reported the problem. Look through the [existing issues](<LINK_TO_GITHUB_ISSUES>) for similar issues.

Report a bug by submitting a [bug report](<LINK_TO_CREATE_BUG_REPORT>). Make sure that you provide as much information as possible on how to reproduce the bug.

When opening a Bug please include the following information to help with debugging:

1. Version of relevant software
2. Details of the issue explaining the problem: what, when, where
3. The expected outcome that was not met (if any)
4. Supporting troubleshooting information. __Note: Do not provide private company information that could compromise your company's security.__

An Issue __must__ be created before submitting any pull request. Any pull request that is created should be linked to an Issue.

__Note: The code and documentation are released with no warranties or SLAs and are intended to be supported through a community driven process.

### Feature request

If you have an idea of how to improve this project, submit a [feature request](<LINK_TO_CREATE_FEATURE_REQUEST_ISSUE>).

### Answering questions

If you have a question and you can't find the answer in the documentation or issues, the next step is to submit a [question.](<LINK_TO_CREATE_QUESTION_ISSUE>)

We'd love your help answering questions being asked by other CSM users.

## Triage issues

Triage helps ensure that issues resolve quickly by:

* Ensuring the issue's intent and purpose is conveyed precisely. This is necessary because it can be difficult for an issue to explain how an end user experiences a problem and what actions they took.
* Giving a contributor the information they need before they commit to resolving an issue.
* Lowering the issue count by preventing duplicate issues.
* Streamlining the development process by preventing duplicate discussions.

If you don't have the knowledge or time to code, consider helping with _issue triage_. The CERT-CSI community will thank you for saving them time by spending some of yours.

Read more about the ways you can [Triage issues](ISSUE_TRIAGE.md).

## Your first contribution

Unsure where to begin contributing? Start by browsing issues labeled `beginner friendly` or `help wanted`.

* [Beginner-friendly](<LINK_TO_GITHUB_ISSUES_LABELLED_BEGINNER_FRIENDLY>) issues are generally straightforward to complete.
* [Help wanted](<LINK_TO_GITHUB_ISSUES_LABELLED_HELP_WANTED>) issues are problems we would like the community to help us with regardless of complexity.

When you're ready to contribute, it's time to create a pull request.

## Branching

CERT-CSI follows a scaled trunk branching strategy where short-lived branches are created off of the main branch. When coding is complete, the branch is merged back into main after being approved in a pull request code review.

### Steps for branching and contributing

1. Fork the repository.
2. Create a branch off of the main branch.
3. Make your changes and commit them to your branch.
4. If other code changes have merged into the upstream main branch, perform a rebase of those changes into your branch.
5. Open a [pull request](<LINK_TO_CREATE_PULL_REQUEST>) between your branch and the upstream main branch.
6. Once your pull request has merged, your branch can be deleted.

## Signing your commits

We require that developers sign off their commits to certify that they have permission to contribute the code in a pull request. This way of certifying is commonly known as the [Developer Certificate of Origin (DCO)](https://developercertificate.org/). We encourage all contributors to read the DCO text before signing a commit and making contributions.

GitHub will prevent a pull request from being merged if there are any unsigned commits.

### Signing a commit

GPG (GNU Privacy Guard) will be used to sign commits.  Follow the instructions [here](https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/signing-commits) to create a GPG key and configure your GitHub account to use that key.

Make sure you have your user name and e-mail set.  This will be required for your signed commit to be properly verified.  Check the following references:

* Setting up your github user name [reference](https://help.github.com/articles/setting-your-username-in-git/)
* Setting up your e-mail address [reference](https://help.github.com/articles/setting-your-commit-email-address-in-git/)

Once Git and your GitHub account have been properly configured, you can add the -S flag to the git commits:

```console
$ git commit -S -m your commit message
# Creates a signed commit
```

### Commit message format

CERT-CSI uses the guidelines for commit messages outlined in [How to Write a Git Commit Message](https://chris.beams.io/posts/git-commit/)

## Pull Requests

If this is your first time contributing to an open-source project on GitHub, make sure you read about [Creating a pull request](https://help.github.com/en/articles/creating-a-pull-request).

A pull request must always link to at least one GitHub issue. If that is not the case, create a GitHub issue and link it.

To increase the chance of having your pull request accepted, make sure your pull request follows these guidelines:

* Title and description matches the implementation.
* Commits within the pull request follow the formatting guidelines.
* The pull request closes one related issue.
* The pull request contains necessary tests that verify the intended behavior.
* If your pull request has conflicts, rebase your branch onto the main branch.

If the pull request fixes a bug:

* The pull request description must include `Fixes #<issue number>`.
* To avoid regressions, the pull request should include tests that replicate the fixed bug.

The CERT-CSI team _squashes_ all commits into one when we accept a pull request. The title of the pull request becomes the subject line of the squashed commit message. We still encourage contributors to write informative commit messages, as they becomes a part of the Git commit body.

We use the pull request title when we generate change logs for releases. As such, we strive to make the title as informative as possible.

Make sure that the title for your pull request uses the same format as the subject line in the commit message.

## Code Reviews

All submissions, including submissions by project members, require review. We use GitHub pull requests for this purpose. Consult [GitHub Help](https://help.github.com/articles/about-pull-requests/) for more information on using pull requests.

A pull request must satisfy following for it to be merged:

* A pull request will require at least 2 maintainer approvals, one of which must come from a code owner.
* Maintainers must perform a review to ensure the changes adhere to guidelines laid out in this document.
* If any commits are made after the PR has been approved, the PR approval will automatically be removed and the above process must happen again.

## Code Style

## TODOs in the code
We don't like TODOs in the code. It is best if you sort out all issues you can see with the changes before we check the changes in. 
