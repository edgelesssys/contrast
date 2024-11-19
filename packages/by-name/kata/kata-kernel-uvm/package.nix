# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  fetchurl,
  linuxManualConfig,
  stdenvNoCC,
  fetchzip,
  kata,
}:

let
  configfile = stdenvNoCC.mkDerivation rec {
    pname = "kata-kernel-config-confidential";
    inherit (kata.kata-runtime) version;

    src = fetchzip {
      url = "https://github.com/kata-containers/kata-containers/releases/download/${version}/kata-static-${version}-amd64.tar.xz";
      hash = "sha256-VcbOY86p8VkI6XvdhHfZNnWVHKuMLW7Xj7uzHHDiVsk=";
    };

    postPatch = ''
      config=$(find . -regex '.*/config-[0-9.-]+-confidential')

      # 1. We don't use an initrd.
      # 2. Enable dm-init, so that we can use `dm-mod.create`.
      # 3. Disable module signing to make the build reproducable.
      substituteInPlace $config \
        --replace-fail 'CONFIG_INITRAMFS_SOURCE="initramfs.cpio.gz"' 'CONFIG_INITRAMFS_SOURCE=""' \
        --replace-fail 'CONFIG_MODULE_SIG=y' 'CONFIG_MODULE_SIG=n' \
        --replace-fail '# CONFIG_DM_INIT is not set' 'CONFIG_DM_INIT=y' \
        --replace-fail '# CONFIG_DMIID is not set' 'CONFIG_DMIID=y' \
        --replace-fail '# CONFIG_TMPFS_POSIX_ACL is not set' 'CONFIG_TMPFS_POSIX_ACL=y' \
        --replace-fail '# CONFIG_TMPFS_XATTR is not set' 'CONFIG_TMPFS_XATTR=y' \
        --replace-fail '# CONFIG_EFIVAR_FS is not set' 'CONFIG_EFIVAR_FS=y' \
        --replace-fail '# CONFIG_RD_ZSTD is not set' 'CONFIG_RD_ZSTD=y' \
        --replace-fail '# CONFIG_VFAT_FS is not se' 'CONFIG_VFAT_FS=y' \
        --replace-fail '# CONFIG_NLS_CODEPAGE_437 is not set' 'CONFIG_NLS_CODEPAGE_437=y' \
        --replace-fail '# CONFIG_NLS_ISO8859_1 is not set' 'CONFIG_NLS_ISO8859_1=y' \
        --replace-fail '# CONFIG_ATA is not set' 'CONFIG_ATA=y'

      echo "CONFIG_ATA_PIIX=y" >> $config
    '';

    dontBuild = true;

    installPhase = ''
      runHook preInstall

      cp $config $out

      runHook postInstall
    '';
  };
in

linuxManualConfig rec {
  version = "6.11";

  # See https://github.com/kata-containers/kata-containers/blob/5f11c0f144037d8d8f546c89a0392dcd84fa99e2/versions.yaml#L198-L201
  src = fetchurl {
    url = "https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-${version}.tar.xz";
    hash = "sha256-VdLGwCXrwngQx0jWYyXdW8YB6NMvhYHZ53ZzUpvayy4=";
  };

  kernelPatches = [
    {
      name = "work-around-the-segfault-issue-in-glibc-2-35";
      patch = ./work-around-the-segfault-issue-in-glibc-2-35.patch;
    }
  ];

  inherit configfile;
  allowImportFromDerivation = true;
}
