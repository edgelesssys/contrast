# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  config,
  lib,
  modulesPath,
  ...
}:

{
  imports = [
    (modulesPath + "/profiles/perlless.nix")
    (modulesPath + "/profiles/image-based-appliance.nix")
  ];

  boot.kernelParams = [
    "systemd.verity=yes"
    "selinux=0"
    "pci=realloc,nocrs,assign-busses"
    "systemd.gpt_auto=0"
  ];
  boot.supportedFilesystems = [
    "erofs"
    "vfat"
  ];
  boot.initrd = {
    supportedFilesystems = [
      "erofs"
      "vfat"
    ];
    availableKernelModules = [
      "dm_mod"
      "dm_verity"
      "overlay"
    ];
    services.lvm.enable = true; # For additional udev rules needed by dm-verity.
    systemd = {
      additionalUpstreamUnits = [
        "veritysetup-pre.target"
        "veritysetup.target"
        "remote-veritysetup.target"
      ];
      storePaths = [
        "${config.boot.initrd.systemd.package}/lib/systemd/systemd-veritysetup"
        "${config.boot.initrd.systemd.package}/lib/systemd/system-generators/systemd-veritysetup-generator"
      ];
    };
  };

  fileSystems = {
    "/" = {
      device = "/dev/mapper/root";
      fsType = "erofs";
      options = [ "ro" ];
    };
  }
  # Create tmpfs on directories that need to be writable for activation.
  # TODO(msanft): This needs better support upstream.
  // lib.listToAttrs (
    lib.map
      (path: {
        name = path;
        value = {
          fsType = "tmpfs";
          neededForBoot = true;
        };
      })
      [
        "/var"
        "/bin"
        "/usr/bin"
        "/tmp"
        "/lib"
        "/root"
        "/lib64"
      ]
  );
  # If /etc is read-only, we need to provide the machine-id file as a mount point for systemd.
  # https://www.freedesktop.org/software/systemd/man/256/machine-id.html#Initialization
  environment.etc."machine-id".text = "";

  # Generators create units depending on the environment of the podvm, which is largely controlled by the host.
  # Thus, we only keep the generators we need.
  # Those that remain are listed in the expected units of the systemd-units e2e test, see packages/by-name/kata/systemd-units.
  boot.initrd.systemd.suppressedStorePaths =
    map (name: "${config.boot.initrd.systemd.package}/lib/systemd/system-generators/${name}")
      [
        # keep-sorted start
        "systemd-factory-reset-generator"
        "systemd-gpt-auto-generator"
        "systemd-hibernate-resume-generator"
        # keep-sorted end
      ];
  systemd.generators = lib.genAttrs (
    [
      # keep-sorted start
      "systemd-cryptsetup-generator"
      "systemd-factory-reset-generator"
      "systemd-gpt-auto-generator"
      "systemd-hibernate-resume-generator"
      "systemd-integritysetup-generator"
      "systemd-ssh-generator"
      "systemd-system-update-generator"
      # keep-sorted end
    ]
    # The debug image provides a login on the serial console.
    ++ lib.optional (!config.contrast.debug.enable) "systemd-getty-generator"
  ) (_: "/dev/null");

  networking.firewall.enable = false;

  # We do not require dynamic host configuration.
  # Additionally, dhcpcd could allow for e.g. route manipulation from the host.
  networking.dhcpcd.enable = false;

  systemd.suppressedSystemUnits = [ "systemd-networkd-varlink-metrics.socket" ];

  system.etc.overlay.mutable = false;
  # Use sysusers instead of userborn (which is used by default by the perlless profile).
  # We don't need sophisticated user generation.
  systemd.sysusers.enable = true;
  services.userborn.enable = false;
  users.allowNoPasswordLogin = true;
  # Unnecessary font stuff
  fonts.enableDefaultPackages = false;
  fonts.fontconfig.enable = false;
  # Check that the system does not contain a Nix store path with those strings.
  system.forbiddenDependenciesRegexes = lib.optionals (!config.contrast.debug.enable) [
    "python" # Some of the debug packages need Python.
  ];

  nixpkgs.hostPlatform.system = "x86_64-linux";
  system.stateVersion = "26.05";
}
