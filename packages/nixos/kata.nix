# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.contrast.kata;
in
{
  options.contrast.kata = {
    enable = lib.mkEnableOption "Enable Kata (non-peerpod) support";
  };

  config = lib.mkIf cfg.enable {
    # https://github.com/kata-containers/kata-containers/blob/3.10.1/src/agent/kata-containers.target
    systemd.targets.kata-containers = {
      description = "Kata Containers Agent Target";
      requires = [
        "basic.target"
        "tmp.mount"
        "kata-agent.service"
      ];
      wantedBy = [ "basic.target" ];
      wants = [
        "chronyd.service"
        # https://github.com/kata-containers/kata-containers/blob/5869046d04553c3bd2f16fa1cfb714133050e537/tools/osbuilder/rootfs-builder/rootfs.sh#L712
        "dbus.socket"
      ];
      conflicts = [
        "rescue.service"
        "rescue.target"
      ];
      after = [
        "basic.target"
        "rescue.service"
        "rescue.target"
      ];
      unitConfig.AllowIsolate = true;
    };

    # The DHCP daemon, which is pulled in via `multi-user.target` may overwrite
    # the network settings of the PodVM, which are given by an agent call, and
    # thus should not be overwritten by settings gathered via DHCP.
    # networking.dhcpcd.enable = false;
    # services.nscd.enable = false;
    # system.nssModules = lib.mkForce [ ];

    # # We don't want a virtual console to be set up, as this breaks the vsock-based
    # # debug shell set up by the Kata agent. An interactive session in the podvm is
    # # of no use anyway, as its consoles are not connected to anything besides the
    # # vsock anyway.
    # console.enable = false;
    # systemd.services."systemd-user-sessions".enable = false;
    # systemd.services."reload-systemd-vconsole-setup".enable = false;
    # systemd.services."systemd-logind".enable = false;
    # systemd.services."rescue".enable = false;
    # systemd.targets."rescue".enable = false;
    # systemd.targets."machines".enable = false;
    # systemd.targets."network-online".enable = false;
    # systemd.sockets."nix-daemon".enable = false;

    systemd.services.kata-agent = {
      description = "Kata Containers Agent";
      documentation = [ "https://github.com/kata-containers/kata-containers" ];
      wants = [ "kata-containers.target" ];
      after = [ "systemd-tmpfiles-setup.service" ]; # Not upstream, but required for /etc/resolv.conf bind mount.
      serviceConfig = {
        Type = "exec"; # Not upstream.
        StandardOutput = "tty";
        ExecStart = "${lib.getExe pkgs.kata-agent}";
        LimitNOFILE = 1048576;
        ExecStop = "${pkgs.coreutils}/bin/sync ; ${config.systemd.package}/bin/systemctl --force poweroff";
        FailureAction = "poweroff";
        OOMScoreAdjust = -997;
      };
      # Not upstream
      environment = {
        KATA_AGENT_LOG_LEVEL = "debug";
        OCICRYPT_KEYPROVIDER_CONFIG = builtins.toFile "policy.json" (
          lib.strings.toJSON { default = [ { type = "insecureAcceptAnything"; } ]; }
        );
      };
    };

    fileSystems."/run" = {
      fsType = "tmpfs";
      options = [
        "nodev"
        "nosuid"
        "size=50%"
      ];
      neededForBoot = true;
    };

    # Not used directly, but required for kernel-specific driver builds.
    boot.kernelPackages = pkgs.recurseIntoAttrs (pkgs.linuxPackagesFor pkgs.kata-kernel-uvm);

    boot.initrd = {
      # Don't require TPM2 support. (additional modules)
      systemd.tpm2.enable = false;
      # Don't require any of the hardware modules NixOS includes by default.
      includeDefaultModules = false;
    };

    networking.resolvconf.enable = false;
    systemd.tmpfiles.settings."10-etc-resolvconf"."/etc/resolv.conf".f = {
      group = "root";
      mode = "0755";
      user = "root";
    };

    environment.etc."kata-opa/default-policy.rego".source = "${pkgs.kata-runtime.src}/src/kata-opa/allow-set-policy.rego";
  };
}
