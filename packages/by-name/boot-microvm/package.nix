# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  writeShellApplication,
  qemu,
  OVMF,
}:

writeShellApplication {
  name = "boot-image";
  runtimeInputs = [ qemu ];
  text = ''
    if [ $# -ne 3 ]; then
      echo "Usage: $0 <kernel> <kernel-params-file> <image>" >&2
      exit 1
    fi

    kernel=$1
    # kernelParams=$(cat "$2")
    image=$3

    tmpFile=$(mktemp)
    cp "$image" "$tmpFile"

    qemu-system-x86_64 \
      -enable-kvm \
      -m 3G \
      -nographic \
      -drive if=pflash,format=raw,readonly=on,file=${OVMF.firmware} \
      -drive if=pflash,format=raw,readonly=on,file=${OVMF.variables} \
      -kernel "$kernel" \
      -append "init=/nix/store/x45q9gkzj8wzw952lv2jrsyx8vqdfx1b-nixos-system-nixos-24.11pre-git/init root=/dev/sda1 rootfstype=erofs rootflags=ro console=ttyS0" \
      -device virtio-scsi-pci,id=scsi0,num_queues=4 \
      -device scsi-hd,drive=drive0,bus=scsi0.0,channel=0,scsi-id=0,lun=0 \
      -drive "file=$tmpFile,if=none,id=drive0"
  '';
}
