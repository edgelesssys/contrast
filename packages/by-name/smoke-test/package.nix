# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  nixos,
  pkgs,
  lib,
  writeShellApplication,
  qemu-cc,
  OVMF-SNP,
}:

let

  # TODO(burgerdev): TDX support

  # Avoid shipping all of the unrelated OVMF build outputs (~hundreds of MB).
  ovmf = pkgs.runCommand "OVMF" { } ''
    cp ${OVMF-SNP}/FV/OVMF.fd $out
  '';

  # VM image that will be started with QEMU.
  # Provides kernel, kernel commandline and initrd.
  image = nixos [
    (
      _:
      {
        boot.loader.grub.enable = false;
        boot.initrd.systemd = {
          enable = true;

          # This allows to proceed until the report target.
          suppressedUnits = [
            "initrd-switch-root.service"
            "initrd-switch-root.target"
            "initrd-find-nixos-closure.service"
          ];

          storePaths = [
            pkgs.contrastPkgs.contrast.initializer
          ];

          services = {
            # TODO(burgerdev): Check whether /dev/sev-guest is available in the initrd.
            report = {
              description = "generate TEE evidence and print report";
              wantedBy = [ "initrd.target" ];
              before = [ "initrd.target" ];

              serviceConfig = {
                Type = "oneshot";
                RemainAfterExit = "yes";
                ExecStart = "${lib.getExe pkgs.contrastPkgs.contrast.initializer} report";
                ExecStopPost = "${pkgs.systemd}/bin/systemctl --force --force poweroff";
                # The qemu invocation below results in ttyS0 being printed to the host's stdout.
                # The journal goes somewhere else with the default kernel commandline, so we can
                # get only the report output by redirecting it to ttyS0.
                StandardOutput = "tty";
                StandardError = "tty";
                TTYPath = "/dev/ttyS0";
                # TODO(burgerdev): Below was suggested by LLM, review what it does and whether necessary.
                TTYReset = true;
                TTYVHangup = true;
              };
            };
          };
        };

        system.stateVersion = "26.05";

        # TODO(burgerdev): This just allows for better debugging, we could remove it before merging.
        services.getty.autologinUser = "root";
        boot.initrd.systemd.emergencyAccess = true;
        systemd.enableEmergencyMode = true;

      }
    )
  ];
in

writeShellApplication {
  name = "smoke-test";
  runtimeInputs = [
    qemu-cc
  ];
  text = ''
    # TODO(burgerdev): this only boots a non-confidential VM in order to get the guest processes working.

    kernel="${image.config.system.build.kernel}/bzImage"
    initrd="${image.config.system.build.initialRamdisk}/initrd"

    # TODO(burgerdev): actually use OVMF
    ovmf="${ovmf}"
    : $ovmf
    cmdline="${lib.escapeShellArgs image.config.boot.kernelParams}"

    qemu-system-x86_64 \
      -enable-kvm \
      -m 1024M \
      -smp 1 \
      -machine q35,accel=kvm \
      -cpu host \
      -serial stdio \
      -display none \
      -kernel "$kernel" \
      -initrd "$initrd" \
      -append "$cmdline" # console=ttyS0 for systemd output
  '';
}
