# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  writeShellApplication,
  runCommand,
  nixos,
  jq,
  snphost,
  coreutils,

  contrast,
  OVMF-SNP,
  qemu-cc,
}:

let
  # TODO(davidweisse): TDX support

  # Avoid shipping all of the unrelated OVMF build outputs (~hundreds of MB).
  ovmf = runCommand "OVMF" { } ''
    cp ${OVMF-SNP}/FV/OVMF.fd $out
  '';

  # VM image that will be started with QEMU.
  # Provides kernel, kernel commandline and initrd.
  image = nixos [
    (_: {
      boot.loader.grub.enable = false;
      # Provides /dev/sev-guest and the configfs-tsm report interface.
      boot.initrd.kernelModules = [ "sev-guest" ];
      boot.initrd.systemd = {
        enable = true;

        # There is no root filesystem to switch to, so stay in the initrd and let it reach
        # initrd.target, which pulls in the report service.
        suppressedUnits = [
          "initrd-switch-root.service"
          "initrd-switch-root.target"
          "initrd-find-nixos-closure.service"
        ];

        storePaths = [
          contrast.initializer
        ];

        services = {
          report = {
            description = "generate TEE evidence and print report";
            wantedBy = [ "initrd.target" ];
            before = [ "initrd.target" ];

            # Power off the VM once the report command exits, regardless of its exit code. On
            # success, postStart makes QEMU exit before the poweroff happens.
            unitConfig = {
              SuccessAction = "poweroff-immediate";
              FailureAction = "poweroff-immediate";
            };

            serviceConfig = {
              Type = "oneshot";
              ExecStart = "${lib.getExe contrast.initializer} report";
              # The qemu invocation below forwards ttyS0 to the host's stdout. Without console=ttyS0,
              # kernel and journal output go to /dev/console (tty0) instead, so redirecting the
              # initializer's output to ttyS0 gets us only the report and its logs. TTYPath is needed
              # because "tty" defaults to /dev/console.
              StandardOutput = "tty";
              StandardError = "tty";
              TTYPath = "/dev/ttyS0";
            };

            # Runs only if ExecStart succeeded. Signals success to the host via QEMU's
            # isa-debug-exit device: writing a byte v to its I/O port makes QEMU exit with
            # (v << 1) | 1, here (0x10 << 1) | 1 = 33. Any other exit (plain poweroff, panic with
            # -no-reboot, ...) is treated as failure by the host.
            # See also https://github.com/qemu/qemu/blob/master/hw/misc/debugexit.c
            postStart = ''
              printf '\x10' | dd of=/dev/port bs=1 seek=$((0xf4)) count=1 status=none
            '';
          };
        };
      };

      system.stateVersion = "26.05";
    })
  ];

  kernel = "${image.config.system.build.kernel}/bzImage";
  initrd = "${image.config.system.build.initialRamdisk}/initrd";
  cmdline = lib.concatStringsSep " " image.config.boot.kernelParams;
in

writeShellApplication {
  name = "smoke-test";
  runtimeInputs = [
    coreutils
    jq
    qemu-cc
    snphost
  ];
  text = ''
    # QEMU requires cbitpos to match the host's value exactly. snphost exits non-zero if any of
    # its checks fail, including ones irrelevant for running a CVM, so its exit code is ignored.
    # The JSON may also be missing or invalid if snphost fails early, in which case cbitpos stays empty.
    checks=$(snphost ok --output json || true)
    cbitpos=$(jq -r '.tests[] | select(.name == "C-bit location") | .message' <<<"$checks" 2>/dev/null || true)
    if [[ -z "$cbitpos" ]]; then
      echo "Could not determine the C-bit position. Running \`snphost ok\` shows:" >&2
      snphost ok >&2 || true
      exit 1
    fi

    # The guest signals success via isa-debug-exit, which makes QEMU exit with 33. Every other
    # exit code, including 0 from a plain poweroff, means the report was not generated.
    code=0
    timeout 180 qemu-system-x86_64 \
      -machine q35,accel=kvm,confidential-guest-support=snp \
      -object "sev-snp-guest,id=snp,cbitpos=$cbitpos,reduced-phys-bits=1,kernel-hashes=on" \
      -cpu host \
      -m 1024M \
      -serial stdio \
      -display none \
      -no-reboot \
      -device isa-debug-exit,iobase=0xf4,iosize=0x04 \
      -bios "${ovmf}" \
      -kernel "${kernel}" \
      -initrd "${initrd}" \
      -append "${cmdline}" || code=$? # Add "console=ttyS0" to see kernel and journal output

    if [[ $code -ne 33 ]]; then
      case $code in
        124) echo "QEMU timed out" >&2 ;;
        0) echo "VM shut down without reporting success" >&2 ;;
        *) echo "QEMU exited with code $code" >&2 ;;
      esac
      echo "Running \`snphost ok\` to check if SEV-SNP is initialized correctly:" >&2
      snphost ok >&2 || true
      exit 1
    fi
  '';
}
