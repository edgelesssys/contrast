# Installation & setup

## Setup cluster

Please follow our How-to for [AKS](../howto/cluster-setup/aks.md) or [bare metal](../howto/cluster-setup/bare-metal.md) to make your cluster ready for Contrast.

## Install CLI

Download the Contrast CLI from the latest release:

```bash
curl --proto '=https' --tlsv1.2 -fLo contrast https://github.com/edgelesssys/contrast/releases/latest/download/contrast
```

After that, install the Contrast CLI in your PATH, e.g.:

```bash
sudo install contrast /usr/local/bin/contrast
```
