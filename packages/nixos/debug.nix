# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.contrast.debug;
in

{
  options.contrast.debug = {
    enable = lib.mkEnableOption "Enable the debugging environment";
  };

  config = lib.mkIf cfg.enable {
    environment.systemPackages = with pkgs; [
      busybox
      tpm2-tools
      ncurses
      findutils
      curlMinimal
      util-linux
      coreutils
      strace
      tdx-tools
    ];

    services.getty.autologinUser = "root";

    # If the image is to be booted locally for testing purposes through
    # .#boot-image, or if the image is booted on Peer-pods (on Azure), the
    # console should be ttyS0, as this is what Azure and QEMU expose by default to the
    # user for reading. However, when one builds a Kata image (e.g. for bare-metal), setting
    # console=ttyS0 will break the VM logging (i.e. Kata's "reading guest console"), as this
    # only listens on hvc* TTYs. As we have no indicator on whether an image should be booted locally,
    # we only set console=ttyS0 when the image is a debug-image and on Peer-pods. So for local
    # booting of an image, one needs to remove the optional manually.
    boot.kernelParams = lib.optionals config.contrast.peerpods.enable [ "console=ttyS0" ];

    boot.initrd.systemd.emergencyAccess = true;
    systemd.enableEmergencyMode = true;
  };
}
