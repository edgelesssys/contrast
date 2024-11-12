# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  fetchurl,
  buildLinux,
  ...
}:

buildLinux {
  version = "6.11";
  modDirVersion = "6.11.7";

  src = fetchurl {
    url = "https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-6.11.7.tar.xz";
    sha256 = "sha256-C/XsZEgX15KJIPdjWBMR9b8lipJ1nPLzCYXadDrz67I=";
  };

  structuredExtraConfig = with lib.kernel; {
    AMD_MEM_ENCRYPT = lib.mkForce (option yes);
    DRM_AMDGPU = lib.mkForce (option no);
    DRM_AMDGPU_CIK = lib.mkForce (option no);
    DRM_AMDGPU_SI = lib.mkForce (option no);
    DRM_AMDGPU_USERPTR = lib.mkForce (option no);
    DRM_AMD_DC_FP = lib.mkForce (option no);
    DRM_AMD_DC_SI = lib.mkForce (option no);
    HSA_AMD = lib.mkForce (option no);
    DRM_AMD_ACP = lib.mkForce (option no);
    DRM_AMD_DC_DCN = lib.mkForce (option no);
    DRM_AMD_DC_HDCP = lib.mkForce (option no);
    DRM_AMD_SECURE_DISPLAY = lib.mkForce (option no);
    DRM_AMD_ISP = lib.mkForce (option no);
    HYPERV_AZURE_BLOB = lib.mkForce (option no);
    INTEL_TDX_GUEST = lib.mkForce (option yes);
    DEFAULT_SECURITY_APPARMOR = lib.mkForce (option no);
    DEFAULT_SECURITY_SELINUX = lib.mkForce (option no);

    # Required to be compiled with the kernel to allow booting in
    # direct Linux boot scenarios.
    VIRTIO = lib.mkForce (option yes);
    VIRTIO_PCI = lib.mkForce (option yes);
    VIRTIO_BLK = lib.mkForce (option yes);
    VIRTIO_SCSI = lib.mkForce (option yes);
    VIRTIO_MMIO = lib.mkForce (option yes);
    ATA = lib.mkForce (option yes);
    EROFS_FS = lib.mkForce (option yes);
  };

  extraMeta.branch = "6.11";
}
