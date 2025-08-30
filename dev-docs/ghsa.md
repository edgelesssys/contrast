# GitHub Security Advisories

Vulnerabilities in Contrast are remediated and published using the [GHSA workflow].
The goal of the process is to minimize the time between disclosure and availability of a fixed release, while allowing for collaboration among the team and external contributors (where appropriate).

[GHSA workflow]: https://docs.github.com/en/code-security/security-advisories/working-with-repository-security-advisories/about-repository-security-advisories

## Life of a GHSA

### Create

If a vulnerability is discovered, we create an associated GHSA depending on the source:

1. Vulnerability was found by a team member: The finder directly opens a GHSA and fills in the template.
2. Vulnerability was reported by a third party
   1. by opening a GHSA: a team member reviews the GHSA and either accepts or rejects it.
   2. through another channel (for example, via email): a team member opens the GHSA on the finders behalf and adds credit as appropriate.

After creation, at least the _Impact_ section needs to be properly populated.

### Resolve

We use temporary private forks to develop a security fix, following this process:

1. From the GHSA page, create a temporary fork.
2. Develop a fix for the `main` branch and push it to a branch on the private fork.
   Don't be too specific in the commit message, because it will be publicly visible before the fix can be adopted by users.
3. Run e2e tests on the branch with `tools/run-release-test-matrix/run-tests.sh`, covering as many supported platforms as possible.
   GitHub actions don't run on private forks.
4. Add reviewers and follow regular code review practice until approval.
5. Create a new branch based on the latest release branch and cherry-pick the commits from the first branch.
6. Run e2e tests on this branch, too.
7. Add reviewers and follow regular code review practice until approval.

**NOTE**: At the end of this process, there can only be one PR per target branch.
This is a limitation imposed by GitHub.

### Publish

Follow the checklist below to publish the advisory.

1. Review the report:
   1. _Impact_ section describes prerequisites and consequences of an attack.
   2. _Workaround_ section exists, even if it only states that no workaround is possible.
   3. _Patches_ section describes how the vulnerability was fixed.
   4. _Severity_ calculation uses CVSSv3 and matches the impact description.
2. Merge all PRs on the temporary fork.
3. Create a patch release for the latest released version.
4. From the advisory's drop-down menu, select _Publish Advisory_ and click the button.
   We don't usually request CVE assignments through GitHub.

## FAQ

### What if the vulnerability was already disclosed?

This can happen if the vulnerability is evident in a bug report, for example, or through discussion in open forums.
It sometimes makes sense to use a private fork to avoid further leaks, unless it was the fix itself that accidentally disclosed the vulnerability.
Use your best judgement to trade off between risk and ease of development.
In any case, a GHSA needs to be published, following the [Publish](#publish) section reasonably close.
