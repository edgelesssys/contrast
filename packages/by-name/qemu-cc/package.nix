# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  qemu,
  libaio,
  dtc,
  libtasn1,
  numactl,
  python3Packages,
  gpuSupport ? false,
}:
(qemu.override (_previous: {
  minimal = true;
  enableBlobs = true;
  uringSupport = false;
  numaSupport = true;
  # Only build for x86_64.
  hostCpuOnly = true;
  hostCpuTargets = [ "x86_64-softmmu" ];
})).overrideAttrs
  (previousAttrs: {
    configureFlags = previousAttrs.configureFlags ++ [
      "-Dlinux_aio_path=${libaio}/lib"
      "-Dlinux_fdt_path=${dtc}/lib"
    ];

    # The upstream derivation removes the dtc dependency when minimal is set,
    # but QEMU needs it when not only building usermode emulators.
    # TODO(freax13): Fix this upstream.
    #
    # libtasn1 is also dropped by the minimal build, which leaves CONFIG_TASN1 unset.
    # QEMU 11.0.0's tests/qtest/migration/tls-tests.c references its x509 helpers without guarding them on CONFIG_TASN1.
    # We don't even run that test.
    buildInputs = previousAttrs.buildInputs ++ [
      dtc
      libtasn1
    ];

    nativeBuildInputs = previousAttrs.nativeBuildInputs ++ [ python3Packages.packaging ];

    # numactl ships no pkg-config file, so QEMU's meson locates libnuma with cc.find_library('numa'), which fails in nix because NIX_LDFLAGS -L paths are not included.
    postPatch = (previousAttrs.postPatch or "") + ''
      substituteInPlace meson.build \
        --replace-fail "cc.find_library('numa', has_headers: ['numa.h']," \
                       "cc.find_library('numa', has_headers: ['numa.h'], dirs: ['${lib.getLib numactl}/lib'],"
    '';

    patches = [
      ./0001-avoid-duplicate-definitions.patch
      # Based on https://github.com/NixOS/nixpkgs/pull/300070/commits/96054ca98020df125bb91e5cf49bec107bea051b#diff-7246126ac058898e6da6aadc1e831bb26afe07fa145958e55c5e112dc2c578fd.
      # We applied the same change done to libaio to libfdt as well.
      ./0002-add-options-for-library-paths.patch
      ./0003-increase-min-granularity-for-memfd.patch
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
    ]
    ++ lib.optionals (!gpuSupport) [
      # If we're not building with GPU support, we can omit the PCI-related ACPI tables
      # to achieve stable TDX RTMRs.
      ./0006-i386-omit-some-unneeded-ACPI-tables.patch
    ];
  })
