# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  stdenvNoCC,
  kata,
  OVMF-TDX,
  tdx-measure,

  debug ? false,
}:
let
  image = kata.kata-image;
  inherit (image) dmVerityArgs;
  cmdlineBase = "tsc=reliable no_timer_check rcupdate.rcu_expedited=1 i8042.direct=1 i8042.dumbkbd=1 i8042.nopnp=1 i8042.noaux=1 noreplace-smp reboot=k cryptomgr.notests net.ifnames=0 pci=lastbus=0 root=/dev/vda1 rootflags=ro rootfstype=erofs console=hvc0 console=hvc1 quiet systemd.show_status=false panic=1 nr_cpus=1 selinux=0 systemd.unit=kata-containers.target systemd.mask=systemd-networkd.service systemd.mask=systemd-networkd.socket scsi_mod.scan=none";
  cmdlineBaseDebug = "tsc=reliable no_timer_check rcupdate.rcu_expedited=1 i8042.direct=1 i8042.dumbkbd=1 i8042.nopnp=1 i8042.noaux=1 noreplace-smp reboot=k cryptomgr.notests net.ifnames=0 pci=lastbus=0 root=/dev/vda1 rootflags=ro rootfstype=erofs console=hvc0 console=hvc1 debug systemd.show_status=true systemd.log_level=debug panic=1 nr_cpus=1 selinux=0 systemd.unit=kata-containers.target systemd.mask=systemd-networkd.service systemd.mask=systemd-networkd.socket scsi_mod.scan=none agent.log=debug agent.debug_console agent.debug_console_vport=1026";
  cmdline = "${if debug then cmdlineBaseDebug else cmdlineBase} ${dmVerityArgs}";
in
stdenvNoCC.mkDerivation {
  name = "tdx-launch-digests";
  inherit (image) version;

  dontUnpack = true;

  buildPhase = ''
    mkdir $out

    ${lib.getExe tdx-measure} mrtd -f ${OVMF-TDX}/FV/OVMF.fd > $out/mrtd.hex
    ${lib.getExe tdx-measure} rtmr -f ${OVMF-TDX}/FV/OVMF.fd -k ${kata.kata-kernel-uvm}/bzImage -c '${cmdline}' 0 > $out/rtmr0.hex
    ${lib.getExe tdx-measure} rtmr -f ${OVMF-TDX}/FV/OVMF.fd -k ${kata.kata-kernel-uvm}/bzImage -c '${cmdline}' 1 > $out/rtmr1.hex
    ${lib.getExe tdx-measure} rtmr -f ${OVMF-TDX}/FV/OVMF.fd -k ${kata.kata-kernel-uvm}/bzImage -c '${cmdline}' 2 > $out/rtmr2.hex
    ${lib.getExe tdx-measure} rtmr -f ${OVMF-TDX}/FV/OVMF.fd -k ${kata.kata-kernel-uvm}/bzImage -c '${cmdline}' 3 > $out/rtmr3.hex
  '';
}
