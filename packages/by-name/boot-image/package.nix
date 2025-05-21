# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  qemu,
  OVMF,
}:

writeShellApplication {
  name = "boot-image";
  runtimeInputs = [ qemu ];
  text = ''
    tmpFile=$(mktemp)
    cp "$1" "$tmpFile"
    qemu-system-x86_64 \
      -enable-kvm \
      -m 3G \
      -nographic \
      -drive if=pflash,format=raw,readonly=on,file=${OVMF.firmware} \
      -drive if=pflash,format=raw,readonly=on,file=${OVMF.variables} \
      -drive "format=raw,file=$tmpFile"
  '';
}
