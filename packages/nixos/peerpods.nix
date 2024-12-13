# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.contrast.peerpods;
in
{
  options.contrast.peerpods = {
    enable = lib.mkEnableOption "Enable peer pods support";
  };

  config = lib.mkIf cfg.enable {
    systemd.services.kata-agent = {
      description = "Kata Containers Agent";
      documentation = [
        "https://github.com/confidential-containers/cloud-api-adaptor/blob/main/src/cloud-api-adaptor/podvm/files/etc/systemd/system/kata-agent.service"
      ];
      bindsTo = [ "netns@podns.service" ];
      wants = [ "process-user-data.service" ];
      after = [
        "netns@podns.service"
        "process-user-data.service"
      ];
      wantedBy = [ "multi-user.target" ];
      serviceConfig = {
        Type = "exec"; # Not upstream.
        ExecStartPre = [ "${pkgs.coreutils}/bin/mkdir -p /run/kata-containers" ];
        ExecStart = "${lib.getExe pkgs.kata-agent} --config /run/peerpod/agent-config.toml";
        ExecStopPost = "${lib.getExe pkgs.cloud-api-adaptor.kata-agent-clean} --config /run/peerpod/agent-config.toml";
        SyslogIdentifier = "kata-agent";
      };
      environment = {
        KATA_AGENT_LOG_LEVEL = "debug";
        OCICRYPT_KEYPROVIDER_CONFIG = builtins.toFile "policy.json" (
          lib.strings.toJSON { default = [ { type = "insecureAcceptAnything"; } ]; }
        );
      };
    };

    systemd.services.agent-protocol-forwarder = {
      description = "Agent Protocol Forwarder";
      documentation = [
        "https://github.com/confidential-containers/cloud-api-adaptor/blob/main/src/cloud-api-adaptor/podvm/files/etc/systemd/system/agent-protocol-forwarder.service"
      ];
      wants = [ "kata-agent.service" ];
      after = [ "kata-agent.service" ];
      wantedBy = [ "multi-user.target" ];
      unitConfig = {
        DefaultDependencies = false;
      };
      serviceConfig = {
        Type = "notify";
        ExecStart = lib.strings.concatStringsSep " " [
          "${pkgs.cloud-api-adaptor}/bin/agent-protocol-forwarder"
          "-kata-agent-namespace /run/netns/podns"
          "-kata-agent-socket /run/kata-containers/agent.sock"
        ];
        Restart = "on-failure";
        RestartSec = "5s";
      };
    };

    systemd.services.process-user-data = {
      description = "Pull configuration from metadata service";
      documentation = [
        "https://github.com/confidential-containers/cloud-api-adaptor/blob/main/src/cloud-api-adaptor/podvm/files/etc/systemd/system/process-user-data.service"
      ];
      wants = [ "network-online.target" ];
      after = [ "network-online.target" ];
      wantedBy = [ "multi-user.target" ];
      unitConfig = {
        DefaultDependencies = false;
      };
      serviceConfig = {
        Type = "oneshot";
        ExecStart = "${pkgs.cloud-api-adaptor}/bin/process-user-data provision-files";
        RemainAfterExit = true;
      };
    };

    systemd.services."netns@" = {
      description = "Create a network namespace for pod networking";
      documentation = [
        "https://github.com/confidential-containers/cloud-api-adaptor/blob/main/src/cloud-api-adaptor/podvm/files/etc/systemd/system/netns%40.service"
      ];
      serviceConfig = {
        Type = "oneshot";
        RemainAfterExit = true;
        ExecStartPre = "${pkgs.iproute2}/bin/ip netns add %I";
        ExecStart = "${pkgs.iproute2}/bin/ip netns exec %I ${pkgs.iproute2}/bin/ip link set lo up";
        ExecStop = "${pkgs.iproute2}/bin/ip netns del %I";
      };
    };
    # Contrary to bare-metal, a peer pod needs regular network access and DNS. The default setup
    # with dhcpcd and resolvconf does not play well with the immutable /etc, so we use the full
    # systemd stack instead.
    networking.dhcpcd.enable = false;
    systemd.network.enable = true;
    networking.useNetworkd = true;
    services.resolved.enable = true;

    # The /etc/machine-id should be populated by systemd, but the immutable /etc seems to prevent
    # that. We manually fill this file with `uninitialized` to force first-boot behavior. This is
    # required by systemd-networkd.
    # https://www.man7.org/linux//man-pages/man5/machine-id.5.html#FIRST_BOOT_SEMANTICS
    environment.etc."machine-id".text = "uninitialized";

    environment.etc."kata-opa/default-policy.rego".source = pkgs.cloud-api-adaptor.default-policy;
  };
}
