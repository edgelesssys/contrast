# Contrast Documentation

## Previewing changes locally

The Contrast flake contains a development shell for working on the documentation.

It's automatically activated when you are using `direnv`. Otherwise enter the environment with:

```sh
nix develop .#docs
```

Run a local development server previewing your changes with:

```sh
yarn start
```

Browse to <http://localhost:3000/contrast> and choose the "Next" version in the top right.

## CI integration

- **PR preview** Created by [`docs_publish`](../.github/workflows/docs_publish.yml) on PR.
  This will build the website and push it to the `/pr-preview` directory of the [`gh-pages`](https://github.com/edgelesssys/contrast/tree/gh-pages) branch.
- **Publishing** Deployed by [`docs_publish`](../.github/workflows/docs_publish.yml) on push to main.
  This will build the website and push it to the [`gh-pages` branch](https://github.com/edgelesssys/contrast/tree/gh-pages).
- **Actual deployment** happens through a [GitHub controlled action](https://github.com/edgelesssys/contrast/actions/workflows/pages/pages-build-deployment).
- **Release versioning** happens as part of the [release workflow](../.github/workflows/release.yml)

Check out the [latest deployments](https://github.com/edgelesssys/contrast/deployments) (both main and PR preview).


## Attestation graphics

Here is how you can modify and rebuild the attestation graphics.

Requirements:

* `inkscape`
* [goat](https://github.com/blampe/goat)

1. Edit the ASCII art
2. Render the SVG with goat: `goat -i <graphic>.txt -o <graphic>.svg
3. Export text to path with: `inkscape <graphic>.svg -o --export-plain-svg=attestation-pod.svg --export-text-to-path`
