#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

# This script starts a QEMU TDX VM as Kata would (more or less)
# for debugging purposes.

set -euo pipefail

if [[ -z $1 ]]; then
  echo "Usage: $0 <runtime-name>"
  exit 1
fi

runtime_name=$1

bios="/opt/edgeless/${runtime_name}/tdx/share/OVMF.fd"
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
  -bios)
    shift
    bios="$1"
    shift
    ;;
  *)
    shift
    ;;
  esac
done

base_cmdline='tsc=reliable no_timer_check rcupdate.rcu_expedited=1 i8042.direct=1 i8042.dumbkbd=1 i8042.nopnp=1 i8042.noaux=1 noreplace-smp reboot=k cryptomgr.notests net.ifnames=0 pci=lastbus=0 root=/dev/vda1 rootflags=ro rootfstype=erofs console=hvc0 console=hvc1 debug systemd.show_status=true systemd.log_level=debug panic=1 nr_cpus=1 selinux=0 systemd.unit=kata-containers.target systemd.mask=systemd-networkd.service systemd.mask=systemd-networkd.socket scsi_mod.scan=none agent.log=debug agent.debug_console agent.debug_console_vport=1026'
kata_cmdline=$(tomlq -r '.Hypervisor.qemu.kernel_params' <"/opt/edgeless/${runtime_name}/etc/configuration-qemu-tdx.toml")
extra_cmdline='console=ttyS0 systemd.unit=default.target'

"/opt/edgeless/${runtime_name}/tdx/bin/qemu-system-x86_64" \
  -name sandbox-testing,debug-threads=on \
  -uuid 49ce7d67-eade-4708-a81f-b5b904213207 \
  -machine q35,accel=kvm,kernel_irqchip=split,confidential-guest-support=tdx \
  -cpu host,-vmx-rdseed-exit,pmu=off \
  -m 2148M,slots=10,maxmem=516333M \
  -device pci-bridge,bus=pcie.0,id=pci-bridge-0,chassis_nr=1,shpc=off,addr=2,io-reserve=4k,mem-reserve=1m,pref64-reserve=1m \
  -device virtio-serial-pci,disable-modern=false,id=serial0 \
  -device virtio-blk-pci,disable-modern=false,drive=image-3132ead95475d1bb,config-wce=off,share-rw=on,serial=image-3132ead95475d1bb \
  -drive "id=image-3132ead95475d1bb,file=/opt/edgeless/${runtime_name}/share/kata-containers.img,aio=threads,format=raw,if=none,readonly=on" \
  -device virtio-scsi-pci,id=scsi0,disable-modern=false \
  -object '{"qom-type":"tdx-guest","id":"tdx","mrconfigid":"XGOgbZcHhD3KKCQ1Z4aeLiAYlCQu6/zTrhgQLkAQg/cAAAAAAAAAAAAAAAAAAAAA","quote-generation-socket":{"type":"vsock","cid":"2","port":"4050"}}' \
  -object rng-random,id=rng0,filename=/dev/urandom \
  -device virtio-rng-pci,rng=rng0 \
  -rtc base=utc,driftfix=slew,clock=host \
  -global kvm-pit.lost_tick_policy=discard \
  -vga none \
  -no-user-config \
  -nodefaults \
  -nographic \
  --no-reboot \
  -object memory-backend-ram,id=dimm1,size=2148M \
  -kernel "/opt/edgeless/${runtime_name}/share/kata-kernel" \
  -initrd "/opt/edgeless/${runtime_name}/share/kata-initrd.zst" \
  -append "${base_cmdline} ${kata_cmdline} ${extra_cmdline}" \
  -serial stdio \
  -bios "${bios}" \
  -smp 1,cores=1,threads=1,sockets=1,maxcpus=1
