#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

set -euo pipefail

if [[ -z $1 ]]; then
  echo "Usage: $0 <runtime-name>"
  exit 1
fi

runtime_name=$1

bios="/opt/edgeless/${runtime_name}/tdx/share/OVMF.fd"

"/opt/edgeless/${runtime_name}/bin/qemu-system-x86_64" \
  -machine q35,accel=kvm,kernel_irqchip=split,confidential-guest-support=tdx \
  -cpu host,pmu=off \
  -m 67000M \
  -object '{"qom-type":"tdx-guest","id":"tdx"}' \
  -display none \
  -vga none \
  -nodefaults \
  --no-reboot \
  -kernel "/opt/edgeless/${runtime_name}/share/kata-kernel" \
  -append "earlyprintk=ttyS0 console=ttyS0" \
  -serial stdio \
  -bios "${bios}" \
  -smp 1
