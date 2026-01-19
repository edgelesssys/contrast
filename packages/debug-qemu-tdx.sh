#!/usr/bin/env bash
# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# This script starts a QEMU TDX VM as Kata would (more or less)
# for debugging purposes.

set -euo pipefail

if [[ -z $1 ]]; then
  echo "Usage: $0 <runtime-name>"
  exit 1
fi

runtime_name=$1

bios="/opt/edgeless/${runtime_name}/tdx/share/OVMF.fd"
gpu_count=0
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
  -bios)
    shift
    bios="$1"
    shift
    ;;
  -gpus)
    shift
    gpu_count="$1"
    shift
    ;;
  *)
    shift
    ;;
  esac
done

# cf. https://github.com/canonical/tdx/blob/1c9ca3964b617ed2be13b47869df7663c4bd8e5f/guest-tools/run_td#L72C1-L82C26
gpu_args=()
if [[ $gpu_count -gt 0 ]]; then
  mapfile -t gpus < <(lspci -Dn -d 10de: | awk '/0302/ {print $1}' | head -n "$gpu_count")
  for i in "${!gpus[@]}"; do
    gpu="${gpus[$i]}"
    gpu_args+=(
      -object "iommufd,id=iommufd${i}"
      -device "pcie-root-port,port=${i},chassis=${i},id=pci.${i},bus=pcie.0"
      -device "vfio-pci,host=${gpu},bus=pci.${i},iommufd=iommufd${i}"
    )
  done
fi

base_cmdline='tsc=reliable no_timer_check rcupdate.rcu_expedited=1 i8042.direct=1 i8042.dumbkbd=1 i8042.nopnp=1 i8042.noaux=1 noreplace-smp reboot=k cryptomgr.notests net.ifnames=0 pci=lastbus=0 root=/dev/vda1 rootflags=ro rootfstype=erofs console=hvc0 console=hvc1 debug systemd.show_status=true systemd.log_level=debug panic=1 nr_cpus=1 selinux=0 systemd.unit=kata-containers.target systemd.mask=systemd-networkd.service systemd.mask=systemd-networkd.socket scsi_mod.scan=none systemd.verity=yes lsm=landlock,yama,bpf cgroup_no_v1=all agent.log=debug agent.debug_console agent.debug_console_vport=1026'
kata_cmdline=$(tomlq -r '.Hypervisor.qemu.kernel_params' <"/opt/edgeless/${runtime_name}/etc/configuration-qemu-tdx.toml")
extra_cmdline='console=ttyS0 systemd.unit=default.target'

"/opt/edgeless/${runtime_name}/bin/qemu-system-x86_64" \
  -name sandbox-testing,debug-threads=on \
  -uuid 49ce7d67-eade-4708-a81f-b5b904213207 \
  -machine q35,accel=kvm,kernel_irqchip=split,confidential-guest-support=tdx \
  -cpu host,pmu=off \
  -m 2024M \
  -device pci-bridge,bus=pcie.0,id=pci-bridge-0,chassis_nr=1,shpc=off,addr=2,io-reserve=4k,mem-reserve=1m,pref64-reserve=1m \
  -device virtio-serial-pci,disable-modern=false,id=serial0 \
  -device virtio-blk-pci,disable-modern=false,drive=image-3132ead95475d1bb,config-wce=off,share-rw=on,serial=image-3132ead95475d1bb \
  -drive "id=image-3132ead95475d1bb,file=/opt/edgeless/${runtime_name}/share/kata-containers.img,aio=threads,format=raw,if=none,readonly=on" \
  -device virtio-scsi-pci,id=scsi0,disable-modern=false \
  -object '{"qom-type":"tdx-guest","id":"tdx","mrconfigid":"XGOgbZcHhD3KKCQ1Z4aeLiAYlCQu6/zTrhgQLkAQg/cAAAAAAAAAAAAAAAAAAAAA","quote-generation-socket":{"type":"vsock","cid":"2","port":"4050"}}' \
  "${gpu_args[@]}" \
  -rtc base=utc,driftfix=slew,clock=host \
  -global kvm-pit.lost_tick_policy=discard \
  -vga none \
  -no-user-config \
  -nodefaults \
  -nographic \
  --no-reboot \
  -object memory-backend-ram,id=dimm1,size=2024M \
  -numa node,memdev=dimm1 \
  -kernel "/opt/edgeless/${runtime_name}/share/kata-kernel" \
  -initrd "/opt/edgeless/${runtime_name}/share/kata-initrd.zst" \
  -append "${base_cmdline} ${kata_cmdline} ${extra_cmdline}" \
  -serial stdio \
  -bios "${bios}" \
  -fw_cfg name=opt/ovmf/X-PciMmio64Mb,string=$((524288 * (gpu_count > 0 ? gpu_count : 1))) \
  -smp 1,cores=1,threads=1,sockets=1,maxcpus=1
