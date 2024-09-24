# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  stdenvNoCC,
  kata,
  OVMF-SNP,
  python3Packages,

  debug ? false,
}:

let
  kernel = "${kata.kata-kernel-uvm}/bzImage";
  ovmf-snp = "${OVMF-SNP}/FV/OVMF.fd";
  image = kata.kata-image;
  inherit (image) dmVerityArgs;
  cmdlineBase = "tsc=reliable no_timer_check rcupdate.rcu_expedited=1 i8042.direct=1 i8042.dumbkbd=1 i8042.nopnp=1 i8042.noaux=1 noreplace-smp reboot=k cryptomgr.notests net.ifnames=0 pci=lastbus=0 root=/dev/vda1 rootflags=ro rootfstype=erofs console=hvc0 console=hvc1 quiet systemd.show_status=false panic=1 nr_cpus=1 selinux=0 systemd.unit=kata-containers.target systemd.mask=systemd-networkd.service systemd.mask=systemd-networkd.socket scsi_mod.scan=none";
  cmdlineBaseDebug = "tsc=reliable no_timer_check rcupdate.rcu_expedited=1 i8042.direct=1 i8042.dumbkbd=1 i8042.nopnp=1 i8042.noaux=1 noreplace-smp reboot=k cryptomgr.notests net.ifnames=0 pci=lastbus=0 root=/dev/vda1 rootflags=ro rootfstype=erofs console=hvc0 console=hvc1 debug systemd.show_status=true systemd.log_level=debug panic=1 nr_cpus=1 selinux=0 systemd.unit=kata-containers.target systemd.mask=systemd-networkd.service systemd.mask=systemd-networkd.socket scsi_mod.scan=none agent.log=debug agent.debug_console agent.debug_console_vport=1026";
  cmdline = "${if debug then cmdlineBaseDebug else cmdlineBase} ${dmVerityArgs}";
in

stdenvNoCC.mkDerivation {
  name = "snp-launch-digest${lib.optionalString debug "-debug"}";
  inherit (image) version;

  dontUnpack = true;

  buildPhase = ''
    mkdir $out
    ${lib.getExe python3Packages.sev-snp-measure} \
      --mode snp \
      --ovmf ${ovmf-snp} \
      --vcpus 1 \
      --vcpu-type EPYC-Milan \
      --kernel ${kernel} \
      --append '${cmdline}' \
      --output-format hex > $out/milan.hex
    ${lib.getExe python3Packages.sev-snp-measure} \
      --mode snp \
      --ovmf ${ovmf-snp} \
      --vcpus 1 \
      --vcpu-type EPYC-Genoa \
      --kernel ${kernel} \
      --append '${cmdline}' \
      --output-format hex > $out/genoa.hex
  '';

  passthru = {
    inherit dmVerityArgs;
  };
}
