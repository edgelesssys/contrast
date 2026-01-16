# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  qemu-cc,
  stage0-bin,
  kata,
}:

writeShellApplication {
  name = "run-qemu";
  runtimeInputs = [
    qemu-cc
  ];
  runtimeEnv = {
    STAGE0_BIN = stage0-bin;
    KERNEL = "${kata.image}/bzImage";
    INITRD = "${kata.image}/initrd.zst";
    ROOTFS = "${kata.image}/image-podvm-gpu_1-rc1.raw";
  };
  text = builtins.readFile ./run.sh;
}
