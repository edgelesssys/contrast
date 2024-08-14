# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  stdenvNoCC,
  kata,
  OVMF-SNP,
  python3Packages,
  lib,
}:

let
  kernel = "${kata.kata-kernel-uvm}/bzImage";
  ovmf-snp = "${OVMF-SNP}/FV/OVMF.fd";
  image = kata.kata-image;
  dataBlockSize = builtins.readFile "${image.verity}/data_block_size";
  hashBlockSize = builtins.readFile "${image.verity}/hash_block_size";
  dataBlocks = builtins.readFile "${image.verity}/data_blocks";
  rootHash = builtins.readFile "${image.verity}/roothash";
  salt = builtins.readFile "${image.verity}/salt";
  dataSectorsPerBlock = (lib.strings.toInt dataBlockSize) / 512;
  dataSectors = (lib.strings.toInt dataBlocks) * dataSectorsPerBlock;
  dmVerityArgs = "dm-mod.create=\"dm-verity,,,ro,0 ${toString dataSectors} verity 1 /dev/vda1 /dev/vda2 ${dataBlockSize} ${hashBlockSize} ${dataBlocks} 0 sha256 ${rootHash} ${salt}\" root=/dev/dm-0";
in

stdenvNoCC.mkDerivation {
  name = "snp-launch-digest";
  inherit (image) version;

  dontUnpack = true;

  buildPhase = ''
    mkdir $out
    ${python3Packages.sev-snp-measure}/bin/sev-snp-measure \
      --mode snp \
      --ovmf ${ovmf-snp} \
      --vcpus 1 \
      --vcpu-type EPYC-Milan \
      --kernel ${kernel} \
      --append 'tsc=reliable no_timer_check rcupdate.rcu_expedited=1 i8042.direct=1 i8042.dumbkbd=1 i8042.nopnp=1 i8042.noaux=1 noreplace-smp reboot=k cryptomgr.notests net.ifnames=0 pci=lastbus=0 root=/dev/vda1 rootflags=ro rootfstype=erofs console=hvc0 console=hvc1 quiet systemd.show_status=false panic=1 nr_cpus=1 selinux=0 systemd.unit=kata-containers.target systemd.mask=systemd-networkd.service systemd.mask=systemd-networkd.socket scsi_mod.scan=none ${dmVerityArgs}' \
      --output-format hex > $out/milan.hex
    ${python3Packages.sev-snp-measure}/bin/sev-snp-measure \
      --mode snp \
      --ovmf ${ovmf-snp} \
      --vcpus 1 \
      --vcpu-type EPYC-Genoa \
      --kernel ${kernel} \
      --append 'tsc=reliable no_timer_check rcupdate.rcu_expedited=1 i8042.direct=1 i8042.dumbkbd=1 i8042.nopnp=1 i8042.noaux=1 noreplace-smp reboot=k cryptomgr.notests net.ifnames=0 pci=lastbus=0 root=/dev/vda1 rootflags=ro rootfstype=erofs console=hvc0 console=hvc1 quiet systemd.show_status=false panic=1 nr_cpus=1 selinux=0 systemd.unit=kata-containers.target systemd.mask=systemd-networkd.service systemd.mask=systemd-networkd.socket scsi_mod.scan=none ${dmVerityArgs}' \
      --output-format hex > $out/genoa.hex
  '';

  passthru = {
    inherit dmVerityArgs;
  };
}
