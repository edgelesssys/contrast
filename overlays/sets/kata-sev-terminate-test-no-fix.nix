# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# Testing set: same as kata-sev-terminate-test (forces SEV-ES guest termination
# on boot) but WITHOUT the kata runtime hypervisor-exit-fix patch.
# The QEMU patch (SEV-ES termination -> GUEST_PANICKED) is still active.
#
# Purpose: demonstrate that without the kata fix, a crashed CVM blocks for
# ~2 minutes in the vsock dial timeout before surfacing an error.
# Compare with kata-sev-terminate-test (with fix) where error propagation
# takes < 1 second.
#
# Usage: SET=kata-sev-terminate-test-no-fix just e2e <any-snp-test>

_final: prev: {
  contrastPkgs = prev.contrastPkgs.overrideScope (
    contrastPkgsFinal: contrastPkgsPrev: {
      OVMF-SNP = contrastPkgsPrev.OVMF-SNP.override {
        withForceSevTerminate = true;
      };
      kata = contrastPkgsPrev.kata.overrideScope (
        _kataFinal: kataPrev: {
          runtime = kataPrev.runtime.override {
            withHypervisorExitFix = false;
          };
        }
      );
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
