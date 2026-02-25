# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  buildOciImage,
  dockerTools,
  coordinator,
  initializer,
  node-installer-image,
  nodeinstaller,
  busybox,
  e2fsprogs,
  libuuid,
  iptables-legacy,
  cryptsetup,
  pushOCIDir,
}:

let
  containers = {
    coordinator = buildOciImage {
      name = "coordinator";
      tag = "v${coordinator.version}";
      copyToRoot = [
        busybox
        e2fsprogs # mkfs.ext4
        libuuid # blkid
        iptables-legacy
      ]
      ++ (with dockerTools; [ caCertificates ]);
      config = {
        Cmd = [ "${coordinator}/bin/coordinator" ];
        Env = [
          "PATH=/bin" # Explicitly setting this prevents containerd from setting a default PATH.
          "XTABLES_LOCKFILE=/dev/shm/xtables.lock" # Tells iptables where to create the lock file, since the default path does not exist in our image.
        ];
      };
    };
    initializer = buildOciImage {
      name = "initializer";
      tag = "v${initializer.version}";
      copyToRoot = [
        busybox
        cryptsetup
        e2fsprogs # mkfs.ext4
        libuuid # blkid
        iptables-legacy
      ]
      ++ (with dockerTools; [ caCertificates ]);
      config = {
        # Use Entrypoint so we can append arguments.
        Entrypoint = [ "${initializer}/bin/initializer" ];
        Env = [
          "PATH=/bin" # Explicitly setting this prevents containerd from setting a default PATH.
          "XTABLES_LOCKFILE=/dev/shm/xtables.lock" # Tells iptables where to create the lock file, since the default path does not exist in our image.
        ];
      };
    };
  };
in
containers
// {
  push-node-installer-kata =
    pushOCIDir "push-node-installer-kata" node-installer-image
      "v${nodeinstaller.version}";
  push-node-installer-kata-gpu =
    pushOCIDir "push-node-installer-kata-gpu" node-installer-image.gpu
      "v${nodeinstaller.version}";
}
// lib.concatMapAttrs (name: container: {
  "push-${name}" = pushOCIDir name container.outPath container.meta.tag;
}) containers
