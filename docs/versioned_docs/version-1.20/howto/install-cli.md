# Install CLI

This section provides instructions on how to install the Contrast CLI.

## Applicability

Required for deploying with Contrast.

## Prerequisites

1. [Set up cluster](./cluster-setup/bare-metal.md)

## How-to

Download the Contrast CLI from the latest release and install it in your PATH:

<Tabs queryString="platform">
<TabItem value="linux" label="Linux (x86_64)">

```bash
curl --proto '=https' --tlsv1.2 -fLo contrast https://github.com/edgelesssys/contrast/releases/download/v1.20.0/contrast-x86_64-linux
sudo install contrast /usr/local/bin/contrast
```

</TabItem>
<TabItem value="macos" label="macOS (Apple Silicon)">

```bash
curl --proto '=https' --tlsv1.2 -fLo contrast https://github.com/edgelesssys/contrast/releases/download/v1.20.0/contrast-aarch64-darwin
sudo install contrast /usr/local/bin/contrast
```

:::note
If you download the binary via a web browser instead of `curl`, macOS may show a warning
that the software can't be verified. Remove the quarantine attribute before installing the binary:

```bash
sudo xattr -d com.apple.quarantine contrast
```

:::

</TabItem>
</Tabs>
