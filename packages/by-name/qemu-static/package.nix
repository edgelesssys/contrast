# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  qemu,
  libaio,
  dtc,
  fetchurl,
  python3Packages,
}:
let
  patchedDtc = dtc.overrideAttrs (previousAttrs: {
    patches = previousAttrs.patches ++ [
      # Based on https://github.com/NixOS/nixpkgs/pull/309929/commits/13efe012c484484d48661ce3ad1862a718d1991c.
      # We dropped the change to the output library name from "fdt-so" to "fdt"
      # because it's not entirely clear what this change intended and because
      # this actually breaks the QEMU build.
      ./0001-fix-static-build.patch
    ];
  });
in
(qemu.override (_previous: {
  dtc = patchedDtc;
  minimal = true;
  enableBlobs = true;
  uringSupport = false;
  # Only build for x86_64.
  hostCpuOnly = true;
  hostCpuTargets = [ "x86_64-softmmu" ];
})).overrideAttrs
  (previousAttrs: rec {
    version = "9.1.0-rc4";

    src = fetchurl {
      url = "https://download.qemu.org/qemu-${version}.tar.xz";
      hash = "sha256-gnvAOou9nR+yU67yK4Sa2fM2ZChR8zINoLy12ZROhSw=";
    };

    configureFlags = previousAttrs.configureFlags ++ [
      "-Dlinux_aio_path=${libaio}/lib"
      "-Dlinux_fdt_path=${patchedDtc}/lib"
    ];

    nativeBuildInputs = previousAttrs.nativeBuildInputs ++ [ python3Packages.packaging ];

    patches = [
      ./0001-avoid-duplicate-definitions.patch
      # Based on https://github.com/NixOS/nixpkgs/pull/300070/commits/96054ca98020df125bb91e5cf49bec107bea051b#diff-7246126ac058898e6da6aadc1e831bb26afe07fa145958e55c5e112dc2c578fd.
      # We applied the same change done to libaio to libfdt as well.
      ./0002-add-options-for-library-paths.patch
    ];
  })
