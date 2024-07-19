# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  stdenvNoCC,
  kata,
  OVMF-SNP,
  python3Packages,
}:

let
  kernel = "${kata.kata-kernel-uvm}/bzImage";
  ovmf-snp = "${OVMF-SNP}/FV/OVMF.fd";
in

stdenvNoCC.mkDerivation {
  name = "snp-launch-digest";
  inherit (kata.kata-image) version;

  dontUnpack = true;

  # TODO(freax13): Calculate launch measurements for CPU models other than Genoa.
  buildPhase = ''
    ${python3Packages.sev-snp-measure}/bin/sev-snp-measure \
      --mode snp \
      --ovmf ${ovmf-snp} \
      --vcpus 1 \
      --vcpu-type EPYC-Genoa \
      --kernel ${kernel} \
      --append "tsc=reliable no_timer_check rcupdate.rcu_expedited=1 i8042.direct=1 i8042.dumbkbd=1 i8042.nopnp=1 i8042.noaux=1 noreplace-smp reboot=k cryptomgr.notests net.ifnames=0 pci=lastbus=0 root=/dev/vda1 rootflags=ro rootfstype=erofs console=hvc0 console=hvc1 quiet systemd.show_status=false panic=1 nr_cpus=1 selinux=0 systemd.unit=kata-containers.target systemd.mask=systemd-networkd.service systemd.mask=systemd-networkd.socket scsi_mod.scan=none" \
      --output-format hex > $out
  '';
}
