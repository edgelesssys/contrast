# How to release

## Minor, by promoting nightly

The CI should create a nightly minor draft release, called `v1.x.0-yyyy-mm-dd`, and runs the full test suite against it.
This draft release can be promoted to actual release.

1. Check that in the latest nightly, all checks and tests passed. A draft release is created even if that's not the case, but trying to promote a nightly release where linux/darwin releases, nightly e2es or the release e2e failed, will automatically fail anyway.

2. Sanity-check the latest nightly draft release on GitHub. The tag should match `vX.Y.Z-yyyy-mm-dd` where
  `X.Y.Z` is `version.txt` on main with the `-pre` suffix stripped, and the draft must have artifacts attached.

3. Trigger the promote workflow (it always promotes the most recent completed `release_nightly.yml` run):

    ```sh
    gh workflow run release_promote.yml
    ```

4. Test the binary artifact. The artifacts to review come from the nightly build, not the promote run: open the most recent `release_nightly.yml` run and copy the S3 link from its `Pre-release artifacts` job summary. Send Privatemode a message to review these artifacts and wait for their feedback.

5. **Wait for PM approval before proceeding.**

6. Review the release notes. If label/title/description changes are necessary, change them on the original PR itself, then regenerate the notes on the draft (see [Editing the release notes](#editing-the-release-notes)).

7. Approve the `Publish release` job in the GitHub Actions workflow run.

8. Check that the publish job succeeds.

9. Review and merge the auto generated update PR for `main`. It advertises the new version in `contrast-releases.json` and the docs, so merging it earlier means reverting it if the release fails.

## Editing the release notes

The notes are generated once, when the draft release is created.
Whatever the draft says when you approve `Publish release` is what ships.

Fix the labels and titles on the PRs, then hit _Generate release notes_ on the draft.
Check the _Previous tag_ the UI picked: for a patch it must be the previous patch, so `v1.23.0` for `v1.23.1`.

Edit the body by hand for anything that isn't auto generated, such as PRs from a GHSA temporary private fork.

```markdown
### :warning: Security fixes

* Fixes [GHSA-xxxx-xxxx-xxxx](https://github.com/edgelesssys/contrast/security/advisories/GHSA-xxxx-xxxx-xxxx)
```

The link is dead until the GHSA is published, which is OK.
