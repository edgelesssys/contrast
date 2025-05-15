# Copyright 2024 Edgeless Systems GmbH
# SPDX-License-Identifier: AGPL-3.0-only

{
  lib,
  nixos,
  pkgs,
}:

let
  # 'nixos' uses 'pkgs' from the point in time where nixpkgs function is evaluated. According
  # to the documentation, we should be able to overwrite 'pkgs' by setting nixpkgs.pkgs in
  # the config, but that doesn't seem to work. We use an overlay for now instead.
  # TODO(katexochen): Investigate why the config option doesn't work.
  outerPkgs = pkgs;

  readModulesDir =
    dir:
    lib.pipe (builtins.readDir dir) [
      (lib.filterAttrs (_filename: type: type == "regular"))
      (lib.mapAttrsToList (filename: _type: "${dir}/${filename}"))
    ];
in

lib.makeOverridable (
  args:
  nixos (
    { modulesPath, ... }:

    {
      imports = [
        "${modulesPath}/image/repart.nix"
        "${modulesPath}/system/boot/uki.nix"
      ] ++ readModulesDir ../../nixos;

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

      # TODO(katexochen): imporve, see comment above.
      nixpkgs.overlays = [
        (_self: _super: {
          inherit (outerPkgs)
            cloud-api-adaptor
            pause-bundle
            nvidia-ctk-oci-hook
            nvidia-ctk-with-config
            tdx-tools
            ;
          inherit (outerPkgs.kata)
            kata-agent
            kata-runtime
            kata-kernel-uvm
            ;
        })
      ];

    }
    // args
  )
)
