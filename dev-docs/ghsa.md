# GitHub Security Advisories

Vulnerabilities in Contrast are remediated and published using the [GHSA workflow].
The goal of the process is to minimize the time between disclosure and availability of a fixed release, while allowing for collaboration among the team and external contributors (where appropriate).

This document discusses advisories for security vulnerabilities in Contrast proper.
There may be other reasons to publish advisories, such as critical dependency vulnerabilities or confidential computing concerns.
The process for these may be different and isn't bound to the description below.

[GHSA workflow]: https://docs.github.com/en/code-security/security-advisories/working-with-repository-security-advisories/about-repository-security-advisories

## Life of a GHSA

### Create

If a vulnerability is discovered, we create an associated GHSA depending on the source:

1. Vulnerability was found by a team member: The finder directly opens a GHSA and fills in the template.
2. Vulnerability was reported by a third party
   1. by opening a GHSA: a team member reviews the GHSA and either accepts or rejects it.
   2. through another channel (for example, via email): a team member opens the GHSA on the finders behalf and adds credit as appropriate.

Third parties reporting through GitHub fill in the form defined by [`.github/VULNERABILITY_REPORT.yml`](../.github/VULNERABILITY_REPORT.yml), whose answers the [Publish](#publish) steps consume.
After creation, at least the _Impact_ section needs to be properly populated.
As soon as the overall description is plausible, request a CVE assignment using the button on the GHSA.

### Resolve

We use temporary private forks to develop a security fix, following this process:

1. From the GHSA page, create a temporary fork.
2. Develop a fix for the `main` branch and push it to a branch on the private fork.
   Don't be too specific in the commit message, because it will be publicly visible before the fix can be adopted by users.
3. Run e2e tests on the branch with `tools/run-release-test-matrix/run-tests.sh`, covering as many supported platforms as possible.
   GitHub actions don't run on private forks.
4. Open a pull request targeting `main`.
5. Add reviewers and follow regular code review practice until approval.
6. Create a new branch based on the latest release branch and cherry-pick the commits from the first branch.
7. Run e2e tests on this branch, too.
8. Open a pull request targeting the latest release branch.
9. Add reviewers and follow regular code review practice until approval.

**NOTE**: At the end of this process, there can only be one PR per target branch.
This is a limitation imposed by GitHub.

### Publish

We don't publish a reporter's write-up as the advisory, because the two serve different readers.
A write-up explains how someone found the problem, while an advisory tells users whether they're affected and what to do about it.
So we rewrite the description every time, and the steps below say what we do with each field of the report.

1. Preserve the report.
   Before editing anything, copy the submitted description into a comment on the advisory, wrapped in a `<details>` block so it stays out of the way.
   Comments stay private to the reporter and the collaborators, so the write-up survives the rewrite.

2. Fill in the metadata, using the form answers as input.
   1. _Ecosystem_ is `Other` and the package is `Contrast`, unless the vulnerability is in a supported and versioned Go library.
   2. _Affected versions_ either lists all versions explicitly, or a range including the affected version published last.
      The reporter's entry under _Affected products_ is one data point for that range.
   3. _Patched versions_ contains the version of the patch release about to be created.
      We don't add the next minor release to this field.
   4. _Severity_ uses CVSSv3, calculated from the perspective of the most affected persona.
      The reporter may have named one in the form; decide it yourself when they didn't.
   5. _CWE_ names the weakness, not its consequence.

3. Rewrite the description.
   Keep these sections, in this order, and no others:
   1. `### Impact`, opening with a bold line that names the affected personas and links to [`security.md`](../docs/docs/security.md#protected-personas-in-a-contrast-deployment), then `#### Prerequisites` and `#### Result`.
   2. `### Affected platforms`, only when the issue affects some platforms and not others.
   3. `### Patches`, naming the patch release. No reporter can fill this in.
   4. `### Workarounds`, even when the only content is that no workaround exists.
      The reporter's _Suggested fix or workaround_ belongs here.
   5. `### References` or `### Credits` for links and attribution.

   The reporter's _Summary_ is usually a paragraph, so it feeds the opening paragraph of the impact section, below the persona line and above the prerequisites.
   Write the advisory title yourself: one declarative sentence naming the component and the defect, as in the titles of our published advisories.
   Their _Details_ and _Proof of concept_ are review material: they stay in the comment from step 1 and aren't published.

4. Check the result before publishing.
   1. No `TODO`, and no leftover `Summary`, `Details` or `Proof of concept` heading.
   2. The persona line, the impact text and the CVSS vector agree with each other.
   3. The title reads as a one-line summary of the problem.
   4. The length is in the range of our published advisories, roughly 150 to 300 words.

5. Merge all PRs on the temporary fork.
   You will need to tick the _bypass branch protections_ box. That's okay.

6. Create a [patch release](release.md#patch) for the latest released version.

7. From the advisory's drop-down menu, select _Publish Advisory_ and click the button.

## FAQ

### What if the vulnerability was already disclosed?

This can happen if the vulnerability is evident in a bug report, for example, or through discussion in open forums.
It sometimes makes sense to use a private fork to avoid further leaks, unless it was the fix itself that accidentally disclosed the vulnerability.
Use your best judgement to trade off between risk and ease of development.
In any case, a GHSA needs to be published, following the [Publish](#publish) section reasonably close.

### Do we treat confidential computing vulnerabilities differently?

Vulnerabilities are treated according to the process above, regardless of them affecting guest or host security.
However, it helps users if we include a prominent paragraph in the impact section that explains who is affected.
We usually differentiate between data owner, workload owner and workload operator personas, see [`security.md`](../docs/docs/security.md#protected-personas-in-a-contrast-deployment).
The CVSS score should be calculated from the perspective of the most affected persona.
