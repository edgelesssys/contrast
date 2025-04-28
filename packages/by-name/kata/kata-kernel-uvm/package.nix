# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  fetchurl,
  linuxManualConfig,
  stdenvNoCC,
  fetchzip,
  fetchpatch,
  kata,
  withGPU ? false,
  ... # Required for invocation through `linuxPackagesFor`, which calls this with the `features` argument.
}:

let
  configfile = stdenvNoCC.mkDerivation rec {
    pname = "kata-kernel-config-confidential";
    inherit (kata.kata-runtime) version;

    src = fetchzip {
      url = "https://github.com/kata-containers/kata-containers/releases/download/${version}/kata-static-${version}-amd64.tar.xz";
      hash = "sha256-1sGpBgIfIFHREpB+PM27IVZCMg3ER9mCXf0TJoJbf5s=";
      stripRoot = false;
    };

    postPatch =
      (
        if withGPU then
          ''
            config=$(find . -regex '.*/config-[0-9.-]+-nvidia-gpu-confidential')

            # Enable dm-init, so that we can use `dm-mod.create`.
            # Source: https://github.com/kata-containers/kata-containers/blob/2c6126d3ab708e480b5aad1e7f7adbe22ffaa539/tools/packaging/kernel/configs/fragments/common/confidential_containers/cryptsetup.conf
            cat <<EOF >> $config
            CONFIG_MD=y
            CONFIG_BLK_DEV_DM=y
            CONFIG_DM_INIT=y
            CONFIG_DM_CRYPT=y
            CONFIG_DM_VERITY=y
            CONFIG_DM_INTEGRITY=y
            EOF
          ''
        else
          ''
            config=$(find . -regex '.*/config-[0-9.-]+-confidential')

            # 1. Our kernel build is independent of any initrd.
            # 2. Enable dm-init, so that we can use `dm-mod.create`.
            substituteInPlace $config \
              --replace-fail 'CONFIG_INITRAMFS_SOURCE="initramfs.cpio.gz"' 'CONFIG_INITRAMFS_SOURCE=""' \
              --replace-fail '# CONFIG_DM_INIT is not set' 'CONFIG_DM_INIT=y'
          ''
      )
      + ''
        # 1. Disable module signing to make the build reproducable.
        # 2. Add some options to enable using the kernel in NixOS. (As NixOS has a hard check on
        # whether all modules required for systemd are present, e.g.)
        substituteInPlace $config \
          --replace-fail 'CONFIG_MODULE_SIG=y' 'CONFIG_MODULE_SIG=n' \
          --replace-fail '# CONFIG_DMIID is not set' 'CONFIG_DMIID=y' \
          --replace-fail '# CONFIG_TMPFS_POSIX_ACL is not set' 'CONFIG_TMPFS_POSIX_ACL=y' \
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
  version = "6.12.22";
  modDirVersion = "${version}" + lib.optionalString withGPU "-nvidia-gpu-confidential";

  # See https://github.com/kata-containers/kata-containers/blob/5f11c0f144037d8d8f546c89a0392dcd84fa99e2/versions.yaml#L198-L201
  src = fetchurl {
    url = "https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-${version}.tar.xz";
    hash = "sha256-q0iACrSZhaeNIxiuisXyj9PhI+oXNX7yFJgQWlMzczY=";
  };

  kernelPatches = [
    # Patch prevents containers with unfixed glibc from crashing.
    # Unsure when this can be removed.
    {
      name = "work-around-the-segfault-issue-in-glibc-2-35";
      patch = fetchpatch {
        url = "https://patchwork.ozlabs.org/project/ubuntu-kernel/patch/20230123140233.790103-2-tim.gardner@canonical.com/raw/";
        hash = "sha256-kDW3yqWHxAGzaaM/5mSNoyMa2WZuyWJbMznPTPgfiyo=";
      };
    }
  ];

  inherit configfile;
  allowImportFromDerivation = true;
}
