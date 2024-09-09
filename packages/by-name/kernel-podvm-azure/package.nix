# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  fetchurl,
  buildLinux,
  ...
}:

buildLinux {
  version = "6.2";
  modDirVersion = "6.2.0";

  src = fetchurl {
    url = "https://cdn.confidential.cloud/constellation/kernel/6.2.0-100.constellation/linux-6.2.0-1018-azure.tar.gz";
    sha256 = "sha256-5UKJsAoQUg2UHzz7OPdCZdlvr7neBIm/J5avu2YKA0Q=";
  };

  kernelPatches = [
    {
      name = "0001-azure-fix-sublevel.patch";
      patch = ./0001-azure-fix-sublevel.patch;
    }
  ];

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
    HYPERV_AZURE_BLOB = lib.mkForce (option no);
    INTEL_TDX_GUEST = lib.mkForce (option yes);
    DEFAULT_SECURITY_APPARMOR = lib.mkForce (option no);
    DEFAULT_SECURITY_SELINUX = lib.mkForce (option no);
  };

  extraMeta.branch = "6.2";
}
