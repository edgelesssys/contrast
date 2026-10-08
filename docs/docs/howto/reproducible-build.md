# Reproducible build

This section shows how to build the Contrast CLI from a release tag and check that it's identical to the published binary.

## Applicability

Optional, for verifying that a released CLI was built from the released source.

## Prerequisites

1. [Download the CLI](./install-cli.md) to the current directory as `contrast`.
2. Install [Nix](https://nixos.org/download/) with flakes enabled.
3. On macOS, configure an `x86_64-linux` [remote builder](https://nix.dev/manual/nix/stable/advanced-topics/distributed-builds) for building the Contrast VM images.

## How-to

:::warning

The build takes a long time.
The CLI embeds the reference values of the Contrast VM images.
Computing them requires building the VM images, including the guest firmware and kernel, for every supported platform and their GPU variants from source.

:::

Build the CLI from the release tag and compare it with the published binary:

```bash
version=$(gh release view --repo edgelesssys/contrast --json tagName -q .tagName)
git clone --depth 1 --branch "$version" https://github.com/edgelesssys/contrast contrast-src
cd contrast-src
echo -n "${version#v}" >| version.txt  # the tag carries the next development version
nix build .#base.contrast.cli-release
cmp result/bin/contrast ../contrast
```

`cmp` prints nothing if the binaries are identical.
