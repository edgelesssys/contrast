# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  writeShellApplication,
  qemu,
  OVMF,
}:

# Usage example:
# outPath=$(nix build .#kata.kata-image --print-out-paths); nix run .#boot-microvm -- "${outPath}/bzImage" "${outPath}/initrd" "${outPath}/image-podvm-gpu_1-rc1.raw" "$(nix eval --raw .#kata.kata-image.cmdline)"

writeShellApplication {
  name = "boot-microvm";
  runtimeInputs = [ qemu ];
  text = ''
    if [[ $# -ne 4 ]]; then
      echo "Usage: $0 <kernel> <initrd> <rootfs> <cmdline>";
      exit 1;
    fi

    tmpFile=$(mktemp)
    cp "$3" "$tmpFile"

    qemu-system-x86_64 \
      -enable-kvm \
      -m 3G \
      -nographic \
      -drive if=pflash,format=raw,readonly=on,file=${OVMF.firmware} \
      -drive if=pflash,format=raw,readonly=on,file=${OVMF.variables} \
      -kernel "$1" \
      -initrd "$2" \
      -append "$4" \
      -drive "if=virtio,format=raw,file=$tmpFile" 
  '';
}
