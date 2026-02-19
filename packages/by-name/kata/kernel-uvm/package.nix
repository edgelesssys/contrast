# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  fetchurl,
  linuxManualConfig,
  stdenvNoCC,
  fetchpatch,
  kata,
  withGPU ? false,
  withAMLSandbox ? true,
  ... # Required for invocation through `linuxPackagesFor`, which calls this with the `features` argument.
}:

let
  configfile = stdenvNoCC.mkDerivation {
    pname = "kata-kernel-config-confidential";

    inherit (kata.release-tarball) version;
    src = kata.release-tarball;

    postPatch =
      (
        if withGPU then
          ''
            config=$(find . -regex '.*/config-[0-9.-]+-nvidia-gpu-confidential')

            # 1. Disable module signing to make the build reproducible.
            substituteInPlace $config \
              --replace-fail '# CONFIG_MD is not set' 'CONFIG_MD=y' \
              --replace-fail 'CONFIG_MODULE_SIG=y' 'CONFIG_MODULE_SIG=n'

            # Enable dm-init, so that we can use `dm-mod.create`.
            # Source: https://github.com/kata-containers/kata-containers/blob/2c6126d3ab708e480b5aad1e7f7adbe22ffaa539/tools/packaging/kernel/configs/fragments/common/confidential_containers/cryptsetup.conf
            cat <<EOF >> $config
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
            # 3. NixOS requires capability of loading kernel modules.
            substituteInPlace $config \
              --replace-fail 'CONFIG_INITRAMFS_SOURCE="initramfs.cpio.gz"' 'CONFIG_INITRAMFS_SOURCE=""' \
              --replace-fail '# CONFIG_DM_INIT is not set' 'CONFIG_DM_INIT=y' \
              --replace-fail '# CONFIG_MODULES is not set' 'CONFIG_MODULES=y'
          ''
      )
      + ''
        # 1. Add some options to enable using the kernel in NixOS. (As NixOS has a hard check on
        # whether all modules required for systemd are present, e.g.)
        substituteInPlace $config \
          --replace-fail '# CONFIG_EFIVAR_FS is not set' 'CONFIG_EFIVAR_FS=y' \
          --replace-fail '# CONFIG_RD_ZSTD is not set' 'CONFIG_RD_ZSTD=y' \
          --replace-fail '# CONFIG_VFAT_FS is not se' 'CONFIG_VFAT_FS=y' \
          --replace-fail 'CONFIG_EXPERT=y' '# CONFIG_EXPERT is not set' \
          --replace-fail '# CONFIG_NLS_CODEPAGE_437 is not set' 'CONFIG_NLS_CODEPAGE_437=y' \
          --replace-fail '# CONFIG_NLS_ISO8859_1 is not set' 'CONFIG_NLS_ISO8859_1=y' \
          --replace-fail '# CONFIG_ATA is not set' 'CONFIG_ATA=y'

        cat <<EOF >> $config
        CONFIG_ATA_PIIX=y
        CONFIG_DMIID=y
        CONFIG_ACPI_DEBUG=y
        EOF
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
  version = "6.18.5";
  modDirVersion = "${version}" + lib.optionalString withGPU "-nvidia-gpu-confidential";

  # See https://github.com/kata-containers/kata-containers/blob/5f11c0f144037d8d8f546c89a0392dcd84fa99e2/versions.yaml#L198-L201
  src = fetchurl {
    url = "https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-${version}.tar.xz";
    hash = "sha256-GJ0fQJzvjQ0jQhDgRZUXLfOS+MspfhS0R+2Vcg4v2UA=";
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
  ]
  ++ lib.optional withAMLSandbox {
    name = "drivers-acpi-add-BadAML-sandbox";
    patch = ./0001-drivers-acpi-add-BadAML-sandbox.patch;
  };

  inherit configfile;
  allowImportFromDerivation = true;
}
