# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  qemu,
  libaio,
  dtc,
  runCommand,
  buildPackages,
  libigvm,
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
  # Fetch a range of commits and write them into a single diff file. This
  # function makes it possible to fetch many commits without using fetchPatch
  # for every single commit. `start` and `end` are both inclusive. Changes to
  # submodules are ommited in the diff file.
  # These patches are compatible with the Linux KVM host patches at https://github.com/AMDESE/linux/tree/snp-guest-req-v3.

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
    # Use a newer src.
    #
    # We can't just fetch from git because QEMU release tarballs contain some
    # extra files that QEMU expects to be there at build-time. Instead we
    # invoke QEMU's make-release script, which is how QEMU release tarballs are
    # created. We need to run this in a fixed output derivation.
    src =
      runCommand "qemu-src"
        {
          # TODO(freax13): igvm_master_v4 is likely not fixed.
          version = "igvm_master_v4";
          src = "https://github.com/roy-hopkins/qemu";
          nativeBuildInputs = with buildPackages; [
            git
            meson
            gnutar
            bash
          ];
          env."GIT_SSL_CAINFO" = "${buildPackages.cacert}/etc/ssl/certs/ca-bundle.crt";
          outputHashMode = "recursive";
          outputHashAlgo = "sha256";
          outputHash = "sha256-pJ8Y9HL1i/sLZcUB5FatWSfpw81mWpINEecM30pi0Ks=";
        }
        ''
          # Clone qemu and checkout the release rev.
          git clone $src -b "$version" --single-branch --depth 1 source
          cd source/

          # Fix the release script and run it.
          substituteInPlace ./scripts/make-release \
            --replace-fail "./make_version.sh" "bash ./make_version.sh" \
            --replace-fail "v\''${version}" "\''${version}"
          bash ./scripts/make-release $src "$version"

          # Extract the release tarball into the out directory.
          mkdir $out
          tar -xf qemu-"$version".tar.xz -C "$out" --strip-components=1
        '';

    propagatedBuildInputs =
      builtins.filter (input: input.pname != "texinfo") previousAttrs.propagatedBuildInputs
      ++ [ libigvm ];
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
    patches = [
      ./0001-avoid-duplicate-definitions.patch
      # Based on https://github.com/NixOS/nixpkgs/pull/300070/commits/96054ca98020df125bb91e5cf49bec107bea051b#diff-7246126ac058898e6da6aadc1e831bb26afe07fa145958e55c5e112dc2c578fd.
      # We applied the same change done to libaio to libfdt as well.
      ./0002-add-options-for-library-paths.patch
    ];
  })
