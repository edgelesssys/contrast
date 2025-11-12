# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

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
    guestImagePull = lib.mkEnableOption "Enable guest-based image pulling";
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

    systemd.targets.initdata = {
      description = "Kata Containers Initdata Parsing";
      requires = [
        "initdata-processor.service"
      ];
    };

    systemd.services.initdata-processor = {
      description = "validate and store initdata content";
      after = [ "basic.target" ];
      before = [ "initdata.target" ];
      wants = [ "initdata.target" ];

      serviceConfig = {
        Type = "oneshot";
        RemainAfterExit = "yes";
        ExecStart = lib.getExe pkgs.contrastPkgs.initdata-processor;
      };
    };

    # https://github.com/kata-containers/kata-containers/blob/3.10.1/src/agent/kata-agent.service.in
    systemd.services.kata-agent = {
      description = "Kata Containers Agent";
      documentation = [ "https://github.com/kata-containers/kata-containers" ];
      wants = [ "kata-containers.target" ];
      after = [
        "initdata.target"
        "systemd-tmpfiles-setup.service" # Not upstream, but required for /etc/resolv.conf bind mount.
      ];
      requires = [
        "initdata.target"
      ];
      serviceConfig = {
        Type = "exec"; # Not upstream.
        StandardOutput = "journal+console";
        StandardError = "inherit";
        ExecStart = "${lib.getExe pkgs.contrastPkgs.kata.agent}";
        LimitNOFILE = 1073741824;
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
        KATA_AGENT_POLICY_FILE = "/run/measured-cfg/policy.rego";
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
    boot.kernelPackages = lib.recurseIntoAttrs (
      pkgs.linuxPackagesFor (
        pkgs.contrastPkgs.kata.kernel-uvm.override {
          withGPU = config.contrast.gpu.enable;
        }
      )
    );

    boot.initrd = {
      # Don't require TPM2 support. (additional modules)
      systemd.tpm2.enable = false;
      # Don't require any of the hardware modules NixOS includes by default.
      includeDefaultModules = false;
    };

    networking.resolvconf.enable = false;

    environment.etc."resolv.conf".text =
      "dummy file, to be bind-mounted by the Kata agent when writing network configuration";

    systemd.services.imagepuller = lib.mkIf cfg.guestImagePull {
      description = "Image Puller";
      documentation = [ "https://github.com/edgelesssys/contrast" ];
      after = [
        "network.target"
        "imagestore.service"
        "initdata-processor.service"
      ];
      requires = [
        "initdata-processor.service"
      ];
      wantedBy = [ "kata-agent.service" ];
      serviceConfig = {
        Type = "exec";
        StandardOutput = "journal+console";
        StandardError = "inherit";
        ExecStart = "${lib.getExe pkgs.contrastPkgs.imagepuller}";
        Restart = "always";
        LimitNOFILE = 1048576;
      };
    };

    systemd.services.imagestore = {
      description = "Secure Image Store";
      documentation = [ "https://github.com/edgelesssys/contrast" ];
      wantedBy = [ "kata-agent.service" ];
      path = with pkgs; [
        cryptsetup
        e2fsprogs
        mount
      ];
      serviceConfig = {
        Type = "exec";
        StandardOutput = "journal+console";
        StandardError = "inherit";
        ExecStart = "${lib.getExe pkgs.contrastPkgs.imagestore}";
      };
    };

    systemd.services.deny-incoming-traffic = {
      description = "Deny all incoming connections";

      # We are doing iptables configuration in the unit, so we need the network
      # service to be started. Note that we don't need to wait for network-online.target
      # since we can already add iptables rules before the network is up.
      wants = [ "network.target" ];
      after = [ "network.target" ];

      # This unit must successfully execute and exit before the kata-agent
      # service starts. Otherwise, the kata-agent service will fail to start.
      requiredBy = [ "kata-agent.service" ];
      before = [ "kata-agent.service" ];

      serviceConfig = {
        # oneshot documentation: "the service manager will consider the unit up after the main process exits. It will then start follow-up units."
        # https://www.freedesktop.org/software/systemd/man/latest/systemd.service.html
        Type = "oneshot";
        RemainAfterExit = "yes";
        ExecStart = ''${pkgs.iptables}/bin/iptables-legacy -I INPUT -m conntrack ! --ctstate ESTABLISHED,RELATED -j DROP'';
      };
    };

    systemd.services.contrast-debug-shell = {
      description = "Contrast debug shell acccess for containers";
      after = [
        "initdata.target" # We need to wait for initdata to be written.
        "network.target" # We bind to localhost, so network needs to be up.
      ];
      wantedBy = [ "kata-agent.service" ];
      unitConfig = {
        ConditionPathExists = "/run/measured-cfg/contrast.insecure-debug";
      };
      serviceConfig = {
        Type = "exec";
        ExecStart = "${lib.getExe pkgs.contrastPkgs.debugshell}";
        Restart = "on-failure";
        RestartSec = 5;
        User = "root";
      };
    };
  };
}
