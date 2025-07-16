# How to release

## Minor

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

6. Review the release notes, test the binary artifact.

7. Review and merge the auto generated update PR for main.

8. Publish the GitHub release.

9. Check that the release publish action succeeds.

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
    git push
    ```

5. Trigger the release workflow

    ```sh
    gh workflow run release.yml --ref $(git rev-parse --abbrev-ref HEAD) -f kind=patch -f version="$REL_VER"
    ```

6. Review the release notes, test the binary artifact.

7. Publish the GitHub release.

8. Check that the release publish action succeeds.
