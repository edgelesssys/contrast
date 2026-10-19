# Reproducible build

This section shows how to build the Contrast release artifacts from a release tag and check that they're identical to the published ones.

## Applicability

Optional, for verifying that a Contrast release was built from the released source.

## Prerequisites

1. [Download the CLI](./install-cli.md) to the current directory as `contrast`.
2. Install [Nix](https://nixos.org/download/) with flakes enabled, and the [GitHub CLI](https://cli.github.com/).
3. On macOS, configure an `x86_64-linux` [remote builder](https://nix.dev/manual/nix/stable/advanced-topics/distributed-builds) for building the Linux artifacts.

## How-to

:::warning

The build takes a long time.
The node installer images contain the Contrast VM images, and the CLI embeds their reference values.
This requires building the VM images, including the guest firmware and kernel, for every supported platform and their GPU variants from source.

:::

### Check out the release

```bash
version=$(gh release view --repo edgelesssys/contrast --json tagName -q .tagName)
git clone --depth 1 --branch "$version" https://github.com/edgelesssys/contrast contrast-src
cd contrast-src
echo -n "${version#v}" >| version.txt  # the tag carries the next development version
```

### Container images

Build the container images and record their digests.
The CLI and the deployment files reference the images by these digests.

```bash
image() {
  out=$(nix build --no-link --print-out-paths ".#legacyPackages.x86_64-linux.base.${2}")
  echo "ghcr.io/edgelesssys/contrast/${1}:latest=ghcr.io/edgelesssys/contrast/${1}:${version}@$(jq -r '.manifests[0].digest' "${out}/index.json")"
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

### Deployment files

Generate the Coordinator and runtime deployment files and check them against the release:

```bash
resourcegen() { nix run .#base.contrast.resourcegen -- --image-replacements cli/cmd/assets/image-replacements.txt "$@"; }
resourcegen --add-load-balancers coordinator >| coordinator.yml
for platform in metal-qemu-snp metal-qemu-snp-gpu metal-qemu-tdx metal-qemu-tdx-gpu; do
  resourcegen --namespace contrast-system --add-namespace-object --platform "$platform" runtime >| "runtime-${platform}.yml"
done
for file in coordinator.yml runtime-*.yml; do
  gh release verify-asset "$version" "$file" --repo edgelesssys/contrast
done
```
