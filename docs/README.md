# Contrast Documentation

## Previewing

During edits you can preview your changes using the [`docusaurus`](https://docusaurus.io/docs/installation):

```sh
# requires node >=16.14
npm run start
```

Browse to <http://localhost:3000/contrast> and choose the "Next" version in the top right.

## Publish process

The docs are updated with [publish-docs](../.github/workflows/publish-docs.yml) when pushed on main.

## Release process

1. [Tagging a new version](https://docusaurus.io/docs/next/versioning#tagging-a-new-version)

    ```shell
    npm run docusaurus docs:version X.X
    ```

    When tagging a new version, the document versioning mechanism will:

    Copy the full `docs/docs/` folder contents into a new `docs/versioned_docs/version-[versionName]/` folder.
    Create a versioned sidebars file based from your current sidebar configuration (if it exists) - saved as `docs/versioned_sidebars/version-[versionName]-sidebars.json`.
    Append the new version number to `versions.json`.
