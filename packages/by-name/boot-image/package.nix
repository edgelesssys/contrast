# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  writeShellApplication,
  qemu,
  OVMF,
}:

writeShellApplication {
  name = "boot-microvm";
  runtimeInputs = [ qemu ];
  text = ''
    if [ ! -f "$1/kernel-params" ]; then
      echo "Error: $1/kernel-params not found" >&2
      exit 1
    fi
    
    tmpFile=$(mktemp)
    cp "$1" "$tmpFile"
    qemu-system-x86_64 \
      -enable-kvm \
      -m 3G \
      -nographic \
      -drive if=pflash,format=raw,readonly=on,file=${OVMF.firmware} \
      -drive if=pflash,format=raw,readonly=on,file=${OVMF.variables} \
      -kernel $1/bzImage \
      -append "$(cat $1/kernel-params)" \
      -drive "format=raw,file=$tmpFile"
  '';
}
