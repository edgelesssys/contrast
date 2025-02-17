# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  qemu,
  libaio,
  dtc,
  python3Packages,
  fetchurl,
  fetchzip,
}:
let
  tdxPatches = fetchzip {
    url = "https://launchpadlibrarian.net/744102817/qemu_8.2.2+ds-0ubuntu2+tdx1.0.debian.tar.xz";
    hash = "sha256-ByvNvdGeYJq5tBh8eONU8drQpg1yWolLTf8yoM2VTik=";
  };
in
(qemu.override (_previous: {
  minimal = true;
  enableBlobs = true;
  uringSupport = false;
  # Only build for x86_64.
  hostCpuOnly = true;
  hostCpuTargets = [ "x86_64-softmmu" ];
})).overrideAttrs
  (previousAttrs: rec {
    version = "8.2.2";

    src = fetchurl {
      url = "https://download.qemu.org/qemu-${version}.tar.xz";
      hash = "sha256-hHNGwbgsGlSyw49u29hVSe3rF0MLfU09oSYg4pYrxPM=";
    };

    configureFlags = previousAttrs.configureFlags ++ [
      "-Dlinux_aio_path=${libaio}/lib"
      "-Dlinux_fdt_path=${dtc}/lib"
    ];

    # The upstream derivation removes the dtc dependency when minimal is set,
    # but QEMU needs it when not only building usermode emulators.
    # TODO(freax13): Fix this upstream.
    buildInputs = previousAttrs.buildInputs ++ [ dtc ];

    nativeBuildInputs = previousAttrs.nativeBuildInputs ++ [ python3Packages.packaging ];

    prePatch = ''
      while read patch; do
        patch="''${patch%%#*}"
        if [[ $patch == "" ]]; then
          continue
        fi
        patch -p1 < ${tdxPatches}/patches/$patch
      done < <(cat ${tdxPatches}/patches/series)
    '';

    patches = [
      ./0001-avoid-duplicate-definitions.patch
      # Based on https://github.com/NixOS/nixpkgs/pull/300070/commits/96054ca98020df125bb91e5cf49bec107bea051b#diff-7246126ac058898e6da6aadc1e831bb26afe07fa145958e55c5e112dc2c578fd.
      # We applied the same change done to libaio to libfdt as well.
      ./0002-add-options-for-library-paths.patch
      # Make the generated ACPI tables more deterministic, so that we get a
      # fixed hash for attestation.
      ./0003-i386-omit-some-unneeded-ACPI-tables.patch
      # Load the initrd to a static address to make RTMRs predictable.
      # Both qemu and OVMF patch the linux kernel header with an initrd
      # address that depends on VM size. The patch by qemu is redundant, but
      # ends up being measured into the RTMR by OVMF. Therefore, we replace it
      # with a static value and apply the same value when calculating the
      # RTMRs.
      #
      # References:
      # - https://github.com/tianocore/edk2/blob/523dbb6d597b63181bba85a337d1f53e511f4822/OvmfPkg/Library/LoadLinuxLib/Linux.c#L414
      #   is where OVMF overwrites the initrd address.
      # - https://www.qemu.org/docs/master/specs/fw_cfg.html is how OVMF learns
      #   about the initrd address.
      ./0004-hw-x86-load-initrd-to-static-address.patch
    ];
  })
