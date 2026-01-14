# Install CLI

This section provides instructions on how to install the Contrast CLI.

## Applicability

Required for deploying with Contrast.

## Prerequisites

1. [Set up cluster](./cluster-setup/bare-metal.md)

## How-to

Download the Contrast CLI from the latest release:

```bash
curl --proto '=https' --tlsv1.2 -fLo contrast https://github.com/edgelesssys/contrast/releases/download/v1.16.0/contrast
```

After that, install the Contrast CLI in your PATH, e.g.:

```bash
sudo install contrast /usr/local/bin/contrast
```
