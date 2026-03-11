# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# Set for testing the OVMF ACPI AML validator against BadAML payloads.
# Unlike badaml-vuln (which disables both the kernel AML sandbox and the OVMF
# validator), this set uses the default OVMF with validation enabled to prove
# that the firmware blocks the attack.
#
# The test injects the deadbeef-file.aml payload via qemu-wrapped. The OVMF
# validator rejects the injected SSDT (it contains a SystemMemory
# OperationRegion not in the allowlist) and aborts ACPI table installation,
# which prevents the VM from booting entirely.

_final: prev: {
  contrastPkgs = prev.contrastPkgs.overrideScope (
    contrastPkgsFinal: contrastPkgsPrev: {
      OVMF-SNP = contrastPkgsPrev.OVMF-SNP.override {
        debug = true;
      };
      kata = contrastPkgsPrev.kata.overrideScope (
        _kataFinal: kataPrev: {
          kernel-uvm = kataPrev.kernel-uvm.override {
            # Disable the kernel AML sandbox so the OVMF validator is tested in isolation.
            withAMLSandbox = false;
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
            withSerialLog = true;
          };
        in
        {
          node-installer-image = contrastPrev.node-installer-image.override {
            OVMF-SNP = contrastPkgsFinal.OVMF-SNP;
            withExtraLayers = [
              (contrastPkgsFinal.ociLayerTar {
                files = [
                  {
                    source = "${qemu}/bin/qemu-system-x86_64";
                    destination = "/opt/edgeless/bin/qemu-system-x86_64";
                  }
                  {
                    source = "${qemu}/bin/qemu-system-x86_64-wrapped";
                    destination = "/opt/edgeless/bin/qemu-system-x86_64-wrapped";
                  }
                  {
                    source = "${contrastPkgsFinal.badaml-payload}/deadbeef-file.aml";
                    destination = "/opt/edgeless/bin/payload.aml";
                  }
                ];
              })
            ];
            withExtraInstallFilesConfig = [
              {
                url = "file:///opt/edgeless/bin/qemu-system-x86_64-wrapped";
                path = "/opt/edgeless/@@runtimeName@@/bin/qemu-system-x86_64-wrapped";
                executable = true;
              }
              {
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
