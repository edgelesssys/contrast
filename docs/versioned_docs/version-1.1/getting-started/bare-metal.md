# Prepare a bare metal instance

## Hardware and firmware setup

1. Update your BIOS to a version that supports AMD SEV-SNP. Updating to the latest available version is recommended as newer versions will likely contain security patches for AMD SEV-SNP.
2. Enter BIOS setup to enable SMEE, IOMMU, RMP coverage, and SEV-SNP. Set the SEV-ES ASID Space Limit to a non-zero number (higher is better).
3. Download the latest firmware version for your processor from [AMD](https://www.amd.com/de/developer/sev.html), unpack it, and place it in `/lib/firmware/amd`.

Consult AMD's [Using SEV with AMD EPYC Processors user guide](https://www.amd.com/content/dam/amd/en/documents/epyc-technical-docs/tuning-guides/58207-using-sev-with-amd-epyc-processors.pdf) for more information.

## Kernel Setup

1. Install a kernel with version 6.11 or greater. If you're following this guide before 6.11 has been released, use 6.11-rc3. Don't use 6.11-rc4 - 6.11-rc6 as they contain a regression. 6.11-rc7+ might work.

## K3s Setup

1. Follow the [K3s setup instructions](https://docs.k3s.io/) to create a cluster.
2. Install a block storage provider such as [Longhorn](https://docs.k3s.io/storage#setting-up-longhorn) and mark it as the default storage class.
