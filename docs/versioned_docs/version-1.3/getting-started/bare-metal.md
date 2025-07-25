# Prepare a bare-metal instance

## Hardware and firmware setup

<Tabs queryString="vendor">
<TabItem value="amd" label="AMD SEV-SNP">
1. Update your BIOS to a version that supports AMD SEV-SNP. Updating to the latest available version is recommended as newer versions will likely contain security patches for AMD SEV-SNP.
2. Enter BIOS setup to enable SMEE, IOMMU, RMP coverage, and SEV-SNP. Set the SEV-ES ASID Space Limit to a non-zero number (higher is better).
3. Download the latest firmware version for your processor from [AMD](https://www.amd.com/de/developer/sev.html), unpack it, and place it in `/lib/firmware/amd`.

Consult AMD's
[Using SEV with AMD EPYC Processors user guide](https://www.amd.com/content/dam/amd/en/documents/epyc-technical-docs/tuning-guides/58207-using-sev-with-amd-epyc-processors.pdf)
for more information.
</TabItem>
<TabItem value="intel" label="Intel TDX"> Follow Canonical's instructions on
[setting up Intel TDX in the host's BIOS](https://github.com/canonical/tdx?tab=readme-ov-file#43-enable-intel-tdx-in-the-hosts-bios).
</TabItem>
</Tabs>

## Kernel Setup

<Tabs queryString="vendor">
<TabItem value="amd" label="AMD SEV-SNP">
Install a kernel with version 6.11 or greater. If you're following this guide before 6.11 has been released, use 6.11-rc3. Don't use 6.11-rc4 - 6.11-rc6 as they contain a regression. 6.11-rc7+ might work.
</TabItem>
<TabItem value="intel" label="Intel TDX">
Follow Canonical's instructions on [setting up Intel TDX on Ubuntu 24.04](https://github.com/canonical/tdx?tab=readme-ov-file#41-install-ubuntu-2404-server-image). Note that Contrast currently only supports Intel TDX with Ubuntu 24.04.
</TabItem>
</Tabs>

Increase the `user.max_inotify_instances` sysctl limit by adding
`user.max_inotify_instances=8192` to `/etc/sysctl.d/99-sysctl.conf` and running
`sysctl --system`.

## K3s Setup

1. Follow the [K3s setup instructions](https://docs.k3s.io/) to create a
   cluster.
2. Install a block storage provider such as
   [Longhorn](https://docs.k3s.io/storage#setting-up-longhorn) and mark it as
   the default storage class.
