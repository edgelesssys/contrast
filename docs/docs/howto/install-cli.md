# Install CLI

This section provides instructions on how to install the Contrast CLI.

## Applicability

Required for deploying with Contrast.

## Prerequisites

1. [Set up cluster](./cluster-setup/bare-metal.md)

## How-to

Download the Contrast CLI from the latest release:

<Tabs queryString="platform">
<TabItem value="linux" label="Linux (x86_64)">

```bash
curl --proto '=https' --tlsv1.2 -fLo contrast https://github.com/edgelesssys/contrast/releases/latest/download/contrast-x86_64-linux
```

</TabItem>
<TabItem value="macos" label="macOS (Apple Silicon)">

```bash
curl --proto '=https' --tlsv1.2 -fLo contrast https://github.com/edgelesssys/contrast/releases/latest/download/contrast-aarch64-darwin
```

</TabItem>
</Tabs>

### Verify the release

Check that the binary is the one published with the release, using the [GitHub CLI](https://cli.github.com/):

```bash
version=$(gh release view --repo edgelesssys/contrast --json tagName -q .tagName)
gh release verify-asset "$version" contrast --repo edgelesssys/contrast
```

To also check that the binary was built from the released source, see [Reproducible build](./reproducible-build.md).

### Install the binary

:::note
If you downloaded the binary via a web browser instead of `curl`, macOS may show a warning
that the software can't be verified. Remove the quarantine attribute before installing the binary:

```bash
sudo xattr -d com.apple.quarantine contrast
```

:::

Install the verified binary in your PATH:

```bash
sudo install contrast /usr/local/bin/contrast
```
