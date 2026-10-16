# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  qemu-cc,
  OVMF-SNP,
  pkgs,
  contrastPkgs,
}:

# Usage example:
# outPath=$(nix build .#base.kata.image --print-out-paths); nix run .#base.boot-microvm -- "${outPath}/bzImage" "${outPath}/initrd" "${outPath}/image-podvm-gpu_1-rc1.raw" "$(cat "${outPath}/cmdline")"

let
  # Avoid shipping all of the unrelated OVMF build outputs (~hundreds of MB).
  ovmf = pkgs.runCommand "OVMF" { } ''
    cp ${OVMF-SNP}/FV/OVMF.fd $out
  '';
in

writeShellApplication {
  name = "boot-microvm";
  runtimeInputs = [
    qemu-cc
  ];
  text = ''
    kernel="${contrastPkgs.demo.kernel}/bzImage"
    initrd="${contrastPkgs.demo.initialRamdisk}/initrd"
    rootfs="${contrastPkgs.demo.image}/image-podvm-gpu_1-rc1.raw"
    ovmf="${ovmf}"
    cmdline="${contrastPkgs.demo.cmdline}"

    # TODO(burgerdev): /dev/sev-guest may not be available in the initrd.
    qemu-system-x86_64 \
      -enable-kvm \
      -m 1024M \
      -object memory-backend-ram,id=dimm1,size=1024M \
      -numa node,memdev=dimm1 \
      -smp 1,cores=1,threads=1,sockets=1,maxcpus=1 \
      -machine q35,accel=kvm,kernel_irqchip=split,confidential-guest-support=snp \
      -cpu EPYC-Genoa,pmu=off \
      -object sev-snp-guest,id=snp,cbitpos=51,reduced-phys-bits=1,kernel-hashes=on \
      -serial stdio -display none \
      -bios "$ovmf" \
      -kernel "$kernel" \
      -initrd "$initrd" \
      -drive id=foo,file=$rootfs,format=raw,if=none \
      -device virtio-blk-pci,disable-modern=false,drive=foo,serial=my-little-rootfs \
      -append "$cmdline"
  '';
}
