# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  writeShellApplication,
  osv-scanner,
  vulnix,
  nix,
  jq,
  coreutils,
}:

# Usage: vuln-scan <sbom.cdx.json> [sarif-output-dir]
writeShellApplication {
  name = "vuln-scan";

  runtimeInputs = [
    osv-scanner
    vulnix
    nix
    jq
    coreutils
  ];

  runtimeEnv = {
    OSV_CONFIG = "${../../../../osv-scanner.toml}";
    VULNIX_WHITELIST = "${../../../../vulnix-whitelist.toml}";
    VULNIX_SARIF_JQ = "${./vulnix-to-sarif.jq}";
  };

  text = builtins.readFile ./vuln-scan.sh;
}
