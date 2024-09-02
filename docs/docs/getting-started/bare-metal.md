# Prepare a Bare Metal Instance

## Hardware & Firmware Setup

1. Update your BIOS to a version that supports AMD SEV-SNP. Updating to the latest available version is recommended as newer versions will likely contain security patches for AMD SEV-SNP.
2. Enter BIOS setup to enable SMEE, IOMMU, RMP coverage, and SEV-SNP. Set the SEV-ES ASID Space Limit to a non-zero number (higher is better).
3. Download the latest firmware version for your processor from [AMD](https://www.amd.com/de/developer/sev.html), unpack it, and place it in `/lib/firmware/amd`.

Consult AMD's [Using SEV with AMD EPYC Processors](https://www.amd.com/content/dam/amd/en/documents/epyc-technical-docs/tuning-guides/58207-using-sev-with-amd-epyc-processors.pdf) user guide for more information.

## Kernel Setup

1. Install a kernel with version 6.11 or greater.
2. Increase the `memlock` limit to 8 GiB.

## K3s Setup

1. Follow the [K3s setup instructions](https://docs.k3s.io/) to create a cluster.
2. Install a block storage provider such as [Longhorn](https://docs.k3s.io/storage#setting-up-longhorn) and mark it as the default storage class.
