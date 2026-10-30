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
# Set VENDORED_SBOMS to a directory of per-package SBOMs (e.g. .#base.sbom.cli.vendored) to attribute findings to packages.
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
    NORMALIZE_JQ = "${./normalize.jq}";
    ATTRIBUTE_JQ = "${./attribute.jq}";
    RENDER_JQ = "${./render.jq}";
  };

  text = builtins.readFile ./vuln-scan.sh;
}
