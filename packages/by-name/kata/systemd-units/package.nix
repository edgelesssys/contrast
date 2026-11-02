# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  jq,
  runCommand,
  writers,
  linkFarm,
  kata,
  reachable-units,
  contrastPkgs,
  os-image ? contrastPkgs.contrast.node-installer-image.os-image,
  withDebug ? contrastPkgs.contrast.node-installer-image.withDebug,
}:

let
  config = os-image.nixosConfig;
  initrdCfg = config.boot.initrd.systemd;
  inherit (config.system.build) toplevel;

  # Same as kata.cmdline.make. The roothash of the image doesn't influence the set of units, so we just use a placeholder instead of building the image to obtain it.
  cmdline = lib.concatStringsSep " " (
    kata.runtime.cmdline.prefix withDebug
    ++ os-image.cmdline (lib.strings.replicate 64 "0")
    ++ kata.runtime.cmdline.suffix withDebug
  );

  # The generators in the initrd, taken from its contents as built, i.e. without suppressed store paths.
  initrdGenerators =
    runCommand "initrd-generators"
      {
        nativeBuildInputs = [ jq ];
        inherit (config.system.build.initialRamdisk) contentsJSON;
        passAsFile = [ "contentsJSON" ];
      }
      ''
        mkdir "$out"
        jq --raw-output '.[].source | select(contains("/system-generators/"))' "$contentsJSONPath" |
          while read -r generator; do
            ln -s "$generator" "$out/"
          done
      '';

  # The /etc of the initrd, made up of the files NixOS copies into it.
  initrdEtc = linkFarm "initrd-etc" (
    lib.mapAttrs' (path: file: lib.nameValuePair (lib.removePrefix "/etc/" path) file.source) (
      lib.filterAttrs (path: file: lib.hasPrefix "/etc/" path && file.enable) initrdCfg.contents
    )
  );

  # Restarts the console setup when a virtual console appears.
  vconsoleRule = ''ACTION=="add", SUBSYSTEM=="vtconsole", KERNEL=="vtcon*", RUN+="systemctl --no-block restart systemd-vconsole-setup.service"'';
  # Runs the command a rule stored for removal of the device, see udev(7). No rule of the image stores one.
  removeCmdRule = ''ACTION=="remove", ENV{REMOVE_CMD}!="", RUN+="$env{REMOVE_CMD}"'';
  # Activates LVM volume groups in a transient unit without dependencies, so it starts no units of the image.
  lvmActivateRule =
    systemdRun:
    ''ENV{LVM_VG_NAME_COMPLETE}=="?*", RUN+="${systemdRun} --no-block --property DefaultDependencies=no --unit lvm-activate-$env{LVM_VG_NAME_COMPLETE} lvm vgchange -aay --autoactivation event $env{LVM_VG_NAME_COMPLETE}"'';
  # Configures a TI switch with a program that doesn't exist on NixOS, and that starts no units anyway.
  devlinkRule = ''ENV{ID_NET_DRIVER}=="am65-cpsw-nuss", SUBSYSTEMS=="platform", DRIVERS=="am65-cpsw-nuss", PROGRAM="/usr/sbin/devlink dev param show platform/%b name switch_mode", RESULT=="*cmode runtime value true*", ENV{ID_NET_BRING_UP_BEFORE_JOINING_BRIDGE}="1"'';
  # The command line of initrd-switch-root.service, as printed by systemd.
  switchRootCommand = ''systemctl --no-block switch-root /sysroot "\''${NEW_INIT}"'';

  # The service directories of the system bus, as set by pkgs.makeDBusConf for the given packages.
  dbusServiceDirs = map (pkg: "${pkg}/share/dbus-1/system-services");

  # See the config type in tools/reachable-units for the format.
  initrd = writers.writeJSON "initrd.json" {
    name = "initrd";
    inInitrd = true;
    target = "initrd.target";
    systemd = initrdCfg.package;
    etc = initrdEtc;
    generatorDirs =
      lib.optional (
        initrdCfg.contents ? "/etc/systemd/system-generators"
      ) initrdCfg.contents."/etc/systemd/system-generators".source
      ++ [ initrdGenerators ];
    # Same as the bus configuration of the initrd in nixpkgs, see nixos/modules/services/system/dbus.nix.
    dbusServiceDirs = dbusServiceDirs [
      config.services.dbus.dbusPackage
      initrdCfg.package
    ];
    reviewedCommands = {
      # initrd-cleanup.service, switching to the final system once the initrd is done.
      "systemctl --no-block isolate initrd-switch-root.target" = [ "initrd-switch-root.target" ];
      # initrd-switch-root.service, which replaces the service manager with that of the final system.
      ${switchRootCommand} = [ ];
      ${vconsoleRule} = [ "systemd-vconsole-setup.service" ];
      ${removeCmdRule} = [ ];
      ${lvmActivateRule "systemd-run"} = [ ];
      ${devlinkRule} = [ ];
    };
    # The initrd has no /etc/fstab, only the fstab of the final system, which is mounted at /sysroot.
    env = {
      SYSTEMD_PROC_CMDLINE = cmdline;
      inherit (initrdCfg.managerEnvironment) SYSTEMD_SYSROOT_FSTAB;
    };
  };

  system = writers.writeJSON "system.json" {
    name = "system";
    inInitrd = false;
    target = "kata-containers.target";
    systemd = config.systemd.package;
    etc = "${toplevel}/etc";
    generatorDirs = [
      "${toplevel}/etc/systemd/system-generators"
      "${config.systemd.package}/lib/systemd/system-generators"
    ];
    dbusServiceDirs = dbusServiceDirs config.services.dbus.packages;
    reviewedCommands = {
      # ExecStop of kata-agent.service, which powers off immediately, without starting units.
      "systemctl --force poweroff" = [ ];
      ${vconsoleRule} = [ "systemd-vconsole-setup.service" ];
      ${removeCmdRule} = [ ];
      ${lvmActivateRule "/run/current-system/systemd/bin/systemd-run"} = [ ];
      ${devlinkRule} = [ ];
    }
    // lib.optionalAttrs config.hardware.nvidia-container-toolkit.enable {
      ${''KERNEL=="nvidia", RUN+="systemctl --no-block restart nvidia-container-toolkit-cdi-generator.service"''} =
        [ ];
    };
    env.SYSTEMD_PROC_CMDLINE = cmdline;
  };
in

runCommand "systemd-units"
  {
    nativeBuildInputs = [ reachable-units ];
    passthru = { inherit cmdline; };
  }
  ''
    {
      reachable-units ${initrd}
      echo
      reachable-units ${system}
    } >"$out"
  ''
