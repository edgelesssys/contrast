#!/usr/bin/env bash
# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

echo "Using:" >&2
echo "  STAGE0_BIN: ${STAGE0_BIN}" >&2
echo "  INITRD:     ${INITRD}" >&2
echo "  KERNEL:     ${KERNEL}" >&2
echo "  ROOTFS:     ${ROOTFS}" >&2
echo "" >&2

qemu-system-x86_64 \
  -enable-kvm \
  -cpu host \
  -m 1G \
  -nodefaults \
  -nographic \
  -no-reboot \
  -serial stdio \
  -machine q35 \
  -bios "${STAGE0_BIN}" \
  -kernel "${KERNEL}" \
  -initrd "${INITRD}" \
  -drive file="${ROOTFS}",format=raw,if=virtio,readonly=on \
  -append "root=/dev/vda1 console=ttyS0 earlyprintk=serial,ttyS0,115200"
