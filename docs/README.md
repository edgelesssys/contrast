# Contrast Documentation

## Previewing

The Contrast flake contains a development shell for working on the documentation.

It's automatically activated when you are using `direnv`. Otherwise enter the environment with:

```sh
nix develop .#docs
```

Run a local development server previewing your changes with:

```sh
yarn install
yarn start
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
