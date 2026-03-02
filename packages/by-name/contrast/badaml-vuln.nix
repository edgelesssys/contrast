# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  buildContrast,
  kata,
  qemu-badaml,
  badaml-payload,
  ociLayerTar,
}:

let
  kataOverlay = _final: prev: {
    kernel-uvm = prev.kernel-uvm.override {
      # Deactivate the AML sandbox so the guest is vulnerable to BadAML attack.
      withAMLSandbox = false;
      # Enable ACPI debug logging to make it easier to verify that the attack is working,
      # or debug it if it isn't.
      withACPIDebug = true;
    };
  };
in

buildContrast (
  final: prev: {
    kata = kata.overrideScope (
      _final: prev:
      kataOverlay _final prev
      // {
        image = prev.image.override {
          pkgsOverlay = _final: prev: {
            contrastPkgs = prev.contrastPkgs.overrideScope (
              _final: prev: {
                kata = prev.kata.overrideScope kataOverlay;
              }
            );
          };
        };
      }
    );

    node-installer-image = prev.node-installer-image.override {
      withExtraLayers = [
        (ociLayerTar {
          files = [
            {
              # The wrapper script that replaces the original qemu binary.
              source = "${qemu-badaml}/bin/qemu-system-x86_64";
              destination = "/opt/edgeless/bin/qemu-system-x86_64-foobar";
            }
            {
              # The actual qemu binary that is wrapped by qemu-badaml.
              source = "${qemu-badaml}/bin/qemu-system-x86_64-wrapped";
              destination = "/opt/edgeless/bin/qemu-system-x86_64-wrapped";
            }
            {
              # The AML payload injected by the wrapper script.
              source = "${badaml-payload}/deadbeef-file.aml"; # Modify payload here.
              destination = "/opt/edgeless/share/qemu/payload.aml";
            }
          ];
        })
      ];
      withExtraInstallFilesConfig = [
        {
          # qemu-badaml is a wrapper script that invokes the real qemu-cc binary.
          # The original install entry for qemu-cc will only install the wrapper scripts,
          # so we need to add an additional step to install the actual, wrapped qemu binary.
          url = "file:///opt/edgeless/bin/qemu-system-x86_64-wrapped";
          path = "/opt/edgeless/@@runtimeName@@/bin/qemu-system-x86_64-wrapped";
          executable = true;
        }
      ];
    };

    # Re-evaluate containers so it picks up the overridden node-installer-image.
    containers = final.callPackage ../../contrast/containers.nix { };
  }
)
