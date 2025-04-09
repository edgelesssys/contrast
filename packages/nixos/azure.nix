# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.contrast.azure;

  azure-storage-rules = ''
    # Azure specific rules.
    ACTION!="add|change", GOTO="walinuxagent_end"
    SUBSYSTEM!="block", GOTO="walinuxagent_end"
    ATTRS{ID_VENDOR}!="Msft", GOTO="walinuxagent_end"
    ATTR{ID_MODEL}!="Virtual_Disk", GOTO="walinuxagent_end"

    # Match the known ID parts for root and resource disks.
    ATTRS{device_id}=="?00000000-0000-*", ENV{fabric_name}="root", GOTO="wa_azure_names"
    ATTRS{device_id}=="?00000000-0001-*", ENV{fabric_name}="resource", GOTO="wa_azure_names"

    # Gen2 disk.
    ATTRS{device_id}=="{f8b3781a-1e82-4818-a1c3-63d806ec15bb}", ENV{fabric_scsi_controller}="scsi0", GOTO="azure_datadisk"
    # Create symlinks for data disks attached.
    ATTRS{device_id}=="{f8b3781b-1e82-4818-a1c3-63d806ec15bb}", ENV{fabric_scsi_controller}="scsi1", GOTO="azure_datadisk"
    ATTRS{device_id}=="{f8b3781c-1e82-4818-a1c3-63d806ec15bb}", ENV{fabric_scsi_controller}="scsi2", GOTO="azure_datadisk"
    ATTRS{device_id}=="{f8b3781d-1e82-4818-a1c3-63d806ec15bb}", ENV{fabric_scsi_controller}="scsi3", GOTO="azure_datadisk"
    GOTO="walinuxagent_end"

    # Parse out the fabric name based off of scsi indicators.
    LABEL="azure_datadisk"
    ENV{DEVTYPE}=="partition", PROGRAM="${pkgs.bash}/bin/sh -c '${pkgs.coreutils}/bin/readlink /sys/class/block/%k/../device | ${pkgs.coreutils}/bin/cut -d: -f4'", ENV{fabric_name}="$env{fabric_scsi_controller}/lun$result", TAG+="systemd"
    ENV{DEVTYPE}=="disk", PROGRAM="${pkgs.bash}/bin/sh -c '${pkgs.coreutils}/bin/readlink /sys/class/block/%k/device | ${pkgs.coreutils}/bin/cut -d: -f4'", ENV{fabric_name}="$env{fabric_scsi_controller}/lun$result", TAG+="systemd"

    ENV{fabric_name}=="scsi0/lun0", ENV{fabric_name}="root"
    ENV{fabric_name}=="scsi0/lun1", ENV{fabric_name}="resource"
    # Don't create a symlink for the cd-rom.
    ENV{fabric_name}=="scsi0/lun2", GOTO="walinuxagent_end"

    # Create the symlinks.
    LABEL="wa_azure_names"
    ENV{DEVTYPE}=="disk", SYMLINK+="disk/azure/$env{fabric_name}", TAG+="systemd"
    ENV{DEVTYPE}=="partition", SYMLINK+="disk/azure/$env{fabric_name}-part%n", TAG+="systemd"

    LABEL="walinuxagent_end"
  '';
in

{
  options.contrast.azure = {
    enable = lib.mkEnableOption "Enable Azure specific settings";
  };

  config = lib.mkIf cfg.enable {
    # TODO(burgerdev): find a recent kernel tailored for Azure.

    boot.initrd = {
      kernelModules = [
        "hv_storvsc"
        "hv_netvsc"
        "scsi_transport_fc"
        "hv_utils"
        "hv_vmbus"
        "pci_hyperv"
        "pci_hyperv_intf"
      ];
      # Source https://github.com/Azure/WALinuxAgent/blob/master/config/66-azure-storage.rules,
      # added SYSTEMD tags! TODO(katexochen): upstream?
      services.udev.rules = azure-storage-rules;
    };

    services.udev.extraRules = azure-storage-rules;
    systemd.services.azure-readiness-report = {
      wantedBy = [
        "basic.target"
        "multi-user.target"
      ];
      wants = [ "network-online.target" ];
      after = [ "network-online.target" ];
      description = "Azure Readiness Report";
      serviceConfig = {
        Type = "oneshot";
        RemainAfterExit = "yes";
        ExecStart = "${lib.getExe pkgs.azure-no-agent}";
      };
    };

    systemd.services.setup-nat-for-imds = {
      wantedBy = [ "multi-user.target" ];
      requires = [ "netns@podns.service" ];
      wants = [ "network-online.target" ];
      after = [
        "network-online.target"
        "netns@podns.service"
      ];
      description = "Setup NAT for IMDS";
      serviceConfig = {
        Type = "oneshot";
        RemainAfterExit = "yes";
        # TODO(msanft): Find out why just ordering this after network-online.target
        # isn't sufficient. (Errors with saying that the network is unreachable)
        Restart = "on-failure";
        RestartSec = "5s";
        ExecStart = "${lib.getExe pkgs.cloud-api-adaptor.setup-nat-for-imds}";
      };
    };
  };
}
