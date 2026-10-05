# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# Builds a micro VM image (i.e. rootfs, kernel and kernel cmdline) from a NixOS
# configuration. These components can then be booted in a microVM-fashion
# with QEMU's direct Linux boot feature.
# See: https://qemu-project.gitlab.io/qemu/system/linuxboot.html

{
  stdenvNoCC,
  closureInfo,
  runCommand,

  erofs-utils,
  gnutar,
  jq,
  systemd,
  util-linux,
  pause-bundle,
}:

nixos-config:

let
  makeClosure = paths: closureInfo { rootPaths = paths; };

  directories = runCommand "directories" { } ''
    mkdir -p $out
    cd $out
    mkdir -p bin boot dev etc home lib lib64 mnt opt proc root run srv sys tmp usr/bin var
  '';
in

stdenvNoCC.mkDerivation {
  name = "rootfs";

  dontUnpack = true;

  nativeBuildInputs = [
    erofs-utils
    gnutar
    jq
    systemd
    util-linux
  ];

  buildPhase = ''
    tarball=$(mktemp)
    append() {
      tar -rf "$tarball" --hard-dereference --sort=name --owner=0 --group=0 --numeric-owner "$@"
    }
    append --files-from "${makeClosure nixos-config.system.build.toplevel}/store-paths"
    append -C "${directories}" .
    append -C "${pause-bundle}" .

    erofs=$(mktemp)
    mkfs.erofs --tar=f "$erofs" "$tarball" \
      -b 4096 \
      -U 7e5cd848-5028-4dc8-8e6e-886492f19984 \
      -L rootfs \
      -T 1 \
      --all-root \
      --hard-dereference \
      -x-1

    repart=$(mktemp -d)
    cat >"$repart/10-root.conf" <<EOF
    [Partition]
    Type = root
    Label = root
    Verity = data
    VerityMatchKey = root
    CopyBlocks = $erofs
    EOF
    cat >"$repart/20-verity.conf" <<EOF
    [Partition]
    Type = root-verity
    Label = root-verity
    Verity = hash
    VerityMatchKey = root
    Minimize = best
    EOF

    mkdir -p "$out"

    sectorSize=512 # This needs to match the Kata runtime's setting of virtio-blk-pci/logical_block_size+physical_block_size.
    systemd-repart \
        --definitions="$repart" --empty=create --dry-run=no --json=pretty \
        --architecture=x86-64  --size=auto \
        --sector-size=$sectorSize  --seed=9DB621B2-1B61-4AF6-ACBC-8ADF2D50F04B \
        "$out/image.raw" \
        | jq 'del(.[].file)' \
        | tee "$out/repart-output.json"
    # TODO(burgerdev): repart-output.json has a `file` property that's not reproducible.
  '';
}
