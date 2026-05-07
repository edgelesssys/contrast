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

4. Test the binary artifact. Send Privatemode a message to review the release artifacts as well and wait for their feedback.

5. **Wait for PM approval before proceeding.**

6. Review and merge the auto generated update PR for `main`.

7. Review the release notes. If label/title/description changes are necessary, change them on the original PR itself.

8. Approve the `Publish release` job in the GitHub Actions workflow run.

9. Check that the publish job succeeds and confirm the release notes on GitHub were regenerated against the now-existing tag.

## Minor, manually

If you need to include new changes merged into main since the last successful nightly, you can instead release manually.

1. Ensure all needed PRs were merged.

2. Update [Planned features and limitations](../docs/docs/architecture/features-limitations.md).

3. Export the release you want to make:

    ```sh
    export REL_VER=v0.1.0
    echo "Releasing $REL_VER"
    ```

4. Create a new temporary branch for the release:

    ```sh
    git switch -c "tmp/$REL_VER"
    git push
    ```

5. Trigger the release workflow

    ```sh
    gh workflow run release.yml --ref $(git rev-parse --abbrev-ref HEAD) -f kind=minor -f version="$REL_VER"
    ```

6. Review the release notes. If label/title/description changes are necessary, change them on the PR itself, then regenerate. Ensure the release is based on the latest minor, not patch release. Test the binary artifact.

7. Send Privatemode a message to review the release artifacts and wait for their feedback.

8. **Wait for PM approval before proceeding.**

9. Review and merge the auto generated update PR for main.

10. Approve the `Publish release` job in the GitHub Actions workflow run. This job only becomes available after all e2e tests have passed.

11. Check that the release publish action succeeds.

## Patch

> [!NOTE]
> We do backports by applying backport labels (`backport release/v<minor>`) to PRs that should be backported.
> The backport then happens automatically by the backport action on merge. If you label a PR that was already
> merged, the backport action can be triggered by adding a `/backport` comment. Ensure the backport PR has
> the proper label to gets listed in the release notes.

1. Ensure all needed PRs were backported to the current release branch, and all backport PRs were merged.

2. Export the release you want to make:

    ```sh
    export REL_VER=v0.1.1
    export CUR_VER="$(echo $REL_VER | awk -F. -v OFS=. '{$NF -= 1 ; print}')"
    echo "Releasing $CUR_VER -> $REL_VER"
    ```

3. Checkout the current release branch:

   ```sh
   git switch "release/${REL_VER%.*}"
   git pull
   ```

4. Create a new temporary branch for the release:

    ```sh
    git switch -c "tmp/$REL_VER"
    git push -u origin "tmp/$REL_VER"
    ```

5. Trigger the release workflow

    ```sh
    gh workflow run release.yml --ref $(git rev-parse --abbrev-ref HEAD) -f kind=patch -f version="$REL_VER" --repo edgelesssys/contrast
    ```

6. Review the release notes. If label/title/description changes are necessary, change them on the PR itself, then regenerate. Ensure the release is based on the latest patch release. Test the binary artifact.

7. Send Privatemode a message to review the release artifacts and wait for their feedback.

8. **Wait for PM approval before proceeding.**

9. Review and merge the auto generated update PR for main.

10. Approve the `Publish release` job in the GitHub Actions workflow run. This job only becomes available after all e2e tests have passed.

11. Check that the release publish action succeeds.
