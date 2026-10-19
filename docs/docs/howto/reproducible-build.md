# Reproducible build

This section shows how to build the Contrast release artifacts from a release tag and check that they're identical to the published ones.

## Applicability

Optional, for verifying that a Contrast release was built from the released source.

## Prerequisites

1. [Download the CLI](./install-cli.md) to the current directory as `contrast`.
2. Install [Nix](https://nixos.org/download/) with `nix-command` and `flakes` enabled.
3. Install the [GitHub CLI](https://cli.github.com/) and [`jq`](https://jqlang.org/download/), and log in with `gh auth login`.
4. On macOS, configure an `x86_64-linux` [remote builder](https://nix.dev/manual/nix/stable/advanced-topics/distributed-builds) for building the Linux artifacts.

:::info

Contrast verifies reproducibility weekly by building the node installer images from source on independent Ubuntu 22.04 and 24.04 hosts with the [reproducibility workflow](https://github.com/edgelesssys/contrast/blob/main/.github/workflows/e2e_runtime-reproducibility.yml).
If you also build on Ubuntu 23.10 or later, run the following to let the Nix build sandbox create user namespaces until the next reboot:

```bash
sudo sysctl kernel.apparmor_restrict_unprivileged_userns=0
```

Differences in the Nix version, the build host's filesystem, or the configured binary caches might cause mismatches, so we recommend staying close to that setup.
If an artifact still differs from the release, please [open an issue](https://github.com/edgelesssys/contrast/issues).
:::

## How-to

:::warning

The build takes a long time.
The node installer images contain the Contrast VM images, and the CLI embeds their digests.
This requires building the VM images, including the guest firmware and kernel, for every supported platform and their GPU variants from source.

:::

### Check out the release

```bash
contrast_release=$(gh release view --repo edgelesssys/contrast --json tagName -q .tagName)
git clone --depth 1 --branch "$contrast_release" https://github.com/edgelesssys/contrast contrast-src
cd contrast-src
echo -n "${contrast_release#v}" >| version.txt  # the tag carries the next development version
```

### Container images

Build the container images and record their digests.
The CLI and the deployment files reference the images by these digests.

```bash
image() {
  out=$(nix build --no-link --print-out-paths ".#legacyPackages.x86_64-linux.base.${2}")
  echo "ghcr.io/edgelesssys/contrast/${1}:latest=ghcr.io/edgelesssys/contrast/${1}:${contrast_release}@$(jq -r '.manifests[0].digest' "${out}/index.json")"
}
{
  image coordinator containers.coordinator
  image initializer containers.initializer
  image service-mesh-proxy containers.service-mesh-proxy
  image node-installer-kata contrast.node-installer-image
  image node-installer-kata-gpu contrast.node-installer-image.gpu
  image debugshell containers.debugshell
} >| cli/cmd/assets/image-replacements.txt
```

### CLI

Build the CLI and compare it with the published binary:

```bash
nix build .#base.contrast.cli-release
cmp result/bin/contrast ../contrast
```

`cmp` prints nothing if the binaries are identical.
`nix build` produces the CLI for the platform you run it on, so run this step on macOS to check the macOS CLI.

### Deployment files

Generate the Coordinator and runtime deployment files and check them against the release:

```bash
resourcegen() { nix run .#base.contrast.resourcegen -- --image-replacements cli/cmd/assets/image-replacements.txt "$@"; }
resourcegen --add-load-balancers coordinator >| coordinator.yml
for platform in $(gh release view "$contrast_release" --repo edgelesssys/contrast --json assets -q '.assets[].name | select(startswith("runtime-")) | ltrimstr("runtime-") | rtrimstr(".yml")'); do
  resourcegen --namespace contrast-system --add-namespace-object --platform "$platform" runtime >| "runtime-${platform}.yml"
done
for file in coordinator.yml runtime-*.yml; do
  gh release verify-asset "$contrast_release" "$file" --repo edgelesssys/contrast
done
```
