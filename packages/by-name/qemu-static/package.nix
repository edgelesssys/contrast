# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  qemu,
  libaio,
  dtc,
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

  # Disable a bunch of features we don't need.
  guestAgentSupport = false;
  numaSupport = false;
  seccompSupport = false;
  alsaSupport = false;
  pulseSupport = false;
  pipewireSupport = false;
  sdlSupport = false;
  jackSupport = false;
  gtkSupport = false;
  vncSupport = false;
  smartcardSupport = false;
  spiceSupport = false;
  ncursesSupport = false;
  usbredirSupport = false;
  xenSupport = false;
  cephSupport = false;
  glusterfsSupport = false;
  openGLSupport = false;
  rutabagaSupport = false;
  virglSupport = false;
  libiscsiSupport = false;
  smbdSupport = false;
  tpmSupport = false;
  uringSupport = false;
  canokeySupport = false;
  capstoneSupport = false;
  enableDocs = false;

  # Only build for x86_64.
  hostCpuOnly = true;
  hostCpuTargets = [ "x86_64-softmmu" ];
})).overrideAttrs
  (previousAttrs: {
    propagatedBuildInputs = builtins.filter (
      input: input.pname != "texinfo"
    ) previousAttrs.propagatedBuildInputs;
    configureFlags =
      (
        # By the time overrideAttrs gets to see the attributes, it's too late
        # for dontAddStaticConfigureFlags, so we need to manually filter out
        # the flags.
        builtins.filter (
          flag: flag != "--enable-static" && flag != "--disable-shared"
        ) previousAttrs.configureFlags
      )
      ++ [
        "--static"
        "-Dlinux_aio_path=${libaio}/lib"
        "-Dlinux_fdt_path=${patchedDtc}/lib"
      ];
    patches = previousAttrs.patches ++ [
      ./0001-avoid-duplicate-definitions.patch
      # Based on https://github.com/NixOS/nixpkgs/pull/300070/commits/96054ca98020df125bb91e5cf49bec107bea051b#diff-7246126ac058898e6da6aadc1e831bb26afe07fa145958e55c5e112dc2c578fd.
      # We applied the same change done to libaio to libfdt as well.
      ./0002-add-options-for-library-paths.patch
    ];
  })
