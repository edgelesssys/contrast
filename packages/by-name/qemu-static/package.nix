# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  qemu,
  libaio,
  dtc,
  python3Packages,
}:
(qemu.override (_previous: {
  minimal = true;
  enableBlobs = true;
  uringSupport = false;
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
    buildInputs = previousAttrs.buildInputs ++ [ dtc ];

    nativeBuildInputs = previousAttrs.nativeBuildInputs ++ [ python3Packages.packaging ];

    patches = [
      ./0001-avoid-duplicate-definitions.patch
      # Based on https://github.com/NixOS/nixpkgs/pull/300070/commits/96054ca98020df125bb91e5cf49bec107bea051b#diff-7246126ac058898e6da6aadc1e831bb26afe07fa145958e55c5e112dc2c578fd.
      # We applied the same change done to libaio to libfdt as well.
      ./0002-add-options-for-library-paths.patch
      # Fix needed for a behaviour change in Linux 6.11-rc4.
      # TODO(freax13): Remove this when QEMU 9.1.2 is released.
      ./0003-accel-kvm-check-for-KVM_CAP_READONLY_MEM-on-VM.patch
      # This series allows VFIO to work on SNP.
      ./0004-guest_memfd-Introduce-an-object-to-manage-the-guest-.patch
      ./0005-guest_memfd-Introduce-a-helper-to-notify-the-shared-.patch
      ./0006-KVM-Notify-the-state-change-via-RamDiscardManager-he.patch
      ./0007-memory-Register-the-RamDiscardManager-instance-upon-.patch
      ./0008-guest-memfd-Default-to-discarded-private-in-guest_me.patch
      ./0009-RAMBlock-make-guest_memfd-require-coordinate-discard.patch
      # Fix needed for map large devices using VFIO.
      ./0010-increase-min-granularity-for-memfd.patch
    ];
  })
