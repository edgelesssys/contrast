# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# Testing set: builds OVMF-SNP with a patch that forces an immediate SEV-ES
# guest termination during early boot. This reliably reproduces the
# "kvm_amd: SEV-ES guest requested termination: 0x0:0x0" kernel message
# for testing kata runtime behavior when a CVM crashes.
#
# Usage: SET=kata-sev-terminate-test just e2e <any-snp-test>
# The CVM will crash immediately on boot, every time.

_final: prev: {
  contrastPkgs = prev.contrastPkgs.overrideScope (
    contrastPkgsFinal: contrastPkgsPrev: {
      OVMF-SNP = contrastPkgsPrev.OVMF-SNP.override {
        withForceSevTerminate = true;
      };
      contrast = contrastPkgsPrev.contrast.overrideScope (
        _contrastFinal: contrastPrev: {
          node-installer-image = contrastPrev.node-installer-image.override {
            inherit (contrastPkgsFinal) OVMF-SNP;
          };
        }
      );
    }
  );
}
