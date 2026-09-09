# How to release

Minor and patch releases are both handled exclusively through the nightly pipeline.
`release_nightly.yml` builds the artifacts and runs the full test suite, while `release_promote.yml` turns the resulting draft into a release.

## Minor

`release_nightly.yml` runs nightly on `main` and creates a minor draft release, called `v1.x.0-yyyy-mm-dd`, running the full test suite against it.
This draft release can be promoted to actual release.

1. Check that in the latest nightly, all checks and tests passed. A draft release is created even if that's not the case, but trying to promote a nightly release where linux/darwin releases, nightly e2es or the release e2e failed, will automatically fail anyway.

2. Sanity-check the latest nightly draft release on GitHub. The tag should match `vX.Y.Z-yyyy-mm-dd` where
  `X.Y.Z` is `version.txt` on main with the `-pre` suffix stripped, and the draft must have artifacts attached.

3. Trigger the promote workflow (without a version, it promotes the most recent completed `release_nightly.yml` run on `main`):

    ```sh
    gh workflow run release_promote.yml
    ```

4. Test the binary artifact. The artifacts to review come from the nightly build, not the promote run: open the most recent `release_nightly.yml` run and copy the S3 link from its `Pre-release artifacts` job summary. Send Privatemode a message to review these artifacts and wait for their feedback.

5. **Wait for PM approval before proceeding.**

6. Review the release notes. If label/title/description changes are necessary, change them on the original PR itself, then regenerate the notes on the draft (see [Editing the release notes](#editing-the-release-notes)).

7. Approve the `Publish release` job in the GitHub Actions workflow run.

8. Check that the publish job succeeds.

9. Review and merge the auto generated update PR for `main`. It advertises the new version in `contrast-releases.json` and the docs, so merging it earlier means reverting it if the release fails.

## Patch

> [!NOTE]
> We do backports by applying backport labels (`backport release/v<minor>`) to PRs that should be backported.
> The backport then happens automatically by the backport action on merge. If you label a PR that was already
> merged, the backport action can be triggered by adding a `/backport` comment. Ensure the backport PR has
> the proper label to gets listed in the release notes.

A patch release works exactly like a minor one, except that the nightly runs on the release branch instead of `main`.
After the nightly on `main` finishes, it dispatches a nightly for the newest supported release branch that has unreleased
commits, so a backport merged during the day is built the same night without any manual step.

1. Ensure all needed PRs were backported to the current release branch, and all backport PRs were merged.

2. Wait for the next night, or trigger the patch nightly yourself as described [below](#manual).

3. Check that in that nightly, all checks and tests passed, and that a draft release `vX.Y.Z-yyyy-mm-dd` with artifacts exists.

4. Trigger the promote workflow with the explicit version (without one, it promotes the latest *minor* nightly from `main`):

    ```sh
    gh workflow run release_promote.yml -f version="v1.X.Y" --repo edgelesssys/contrast
    ```

5. Test the binary artifact. The artifacts to review come from the nightly build: open the `release_nightly.yml` run on the release branch and copy the S3 link from its `Pre-release artifacts` job summary. Send Privatemode a message to review these artifacts and wait for their feedback.

6. **Wait for PM approval before proceeding.**

7. Review the release notes. Ensure the release is based on the latest patch release. If label/title/description changes are necessary, change them on the original PR itself, then regenerate the notes on the draft (see [Editing the release notes](#editing-the-release-notes)).

8. Approve the `Publish release` job in the GitHub Actions workflow run.

9. Check that the publish job succeeds.

10. Review and merge the auto generated update PR for `main`.

## Manual

If you find yourself in the situation that you can't wait for the next nightly run, you can always do
```sh
gh workflow run release_nightly.yml --ref main
```
or
```sh
export REL_BRANCH=release/v0.1
gh workflow run release_nightly.yml --ref "$REL_BRANCH" --repo edgelesssys/contrast
```

to start the `release_nightly.yml` workflow manually.

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
