# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

_final: prev: {
  contrastPkgs = prev.contrastPkgs.overrideScope (
    contrastPkgsFinal: contrastPkgsPrev: {
      # Disable OVMF ACPI validation so the injected SSDT is accepted by the firmware.
      OVMF-SNP = contrastPkgsPrev.OVMF-SNP.override {
        withACPIValidation = false;
      };
      OVMF-TDX = contrastPkgsPrev.OVMF-TDX.override {
        withACPIValidation = false;
      };
      kata = contrastPkgsPrev.kata.overrideScope (
        _kataFinal: kataPrev: {
          kernel-uvm = kataPrev.kernel-uvm.override {
            # Deactivate the AML sandbox so the guest is vulnerable to BadAML attack.
            withAMLSandbox = false;
            # Enable ACPI debug logging to make it easier to verify that the attack is working,
            # or debug it if it isn't.
            withACPIDebug = true;
          };
          image = kataPrev.image.override {
            withBadAMLTarget = true;
          };
        }
      );
      contrast = contrastPkgsPrev.contrast.overrideScope (
        _contrastFinal: contrastPrev:
        let
          qemu = contrastPkgsFinal.contrastPkgsStatic.qemu-wrapped.override {
            withACPITable = true;
          };
        in
        {
          node-installer-image = contrastPrev.node-installer-image.override {
            OVMF-SNP = contrastPkgsFinal.OVMF-SNP;
            OVMF-TDX = contrastPkgsFinal.OVMF-TDX;
            withExtraLayers = [
              (contrastPkgsFinal.ociLayerTar {
                files = [
                  {
                    # The wrapper script that replaces the original qemu binary.
                    source = "${qemu}/bin/qemu-system-x86_64";
                    destination = "/opt/edgeless/bin/qemu-system-x86_64";
                  }
                  {
                    # The actual qemu binary that is wrapped by qemu-wrapped.
                    source = "${qemu}/bin/qemu-system-x86_64-wrapped";
                    destination = "/opt/edgeless/bin/qemu-system-x86_64-wrapped";
                  }
                  {
                    # The AML payload injected by the wrapper script.
                    source = "${contrastPkgsFinal.badaml-payload}/deadbeef-file.aml"; # Modify payload here.
                    destination = "/opt/edgeless/bin/payload.aml";
                  }
                ];
              })
            ];
            withExtraInstallFilesConfig = [
              {
                # qemu-wrapped is a wrapper script that invokes the real qemu-cc binary.
                # The original install entry for qemu-cc will only install the wrapper scripts,
                # so we need to add an additional step to install the actual, wrapped qemu binary.
                url = "file:///opt/edgeless/bin/qemu-system-x86_64-wrapped";
                path = "/opt/edgeless/@@runtimeName@@/bin/qemu-system-x86_64-wrapped";
                executable = true;
              }
              {
                # The AML payload injected by the wrapper script.
                url = "file:///opt/edgeless/bin/payload.aml";
                path = "/opt/edgeless/@@runtimeName@@/bin/payload.aml";
              }
            ];
          };
        }
      );
    }
  );
}
