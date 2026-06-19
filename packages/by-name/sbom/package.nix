# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

# Aggregate CycloneDX SBOM (v1.7) for everything contrast builds.
#
# bombon walks the matrix's closure — emitting a component per store path and
# enriching it with license/source metadata from `meta` — and merges the
# language-level SBOMs that packages expose via passthru.bombonVendoredSbom.
# Those are produced purely from build metadata by buildCargoSbom (crate2nix
# graph) and buildGoModuleSbom (cyclonedx-gomod), so a package joins the
# aggregate simply by setting that passthru.
#
# bombon discovers bombonVendoredSbom by walking drvAttrs, but the matrix is a
# linkFarm that references its contents as runtime paths, hiding those passthrus.
# So the packages that carry one are passed explicitly as extraPaths.
{
  lib,
  buildBom,
  matrix,
  contrastPkgs,
}:

buildBom matrix {
  extraPaths = lib.contrast.collectVendoredSbomPackages [
    "contrastPkgsStatic"
    "contrast-releases"
    "matrix"
    "sbom"
    # cli-release only differs from cli in an embedded genpolicy-settings asset,
    # so its vendored SBOM is identical; skip it to avoid a redundant root.
    "cli-release"
  ] contrastPkgs;
}
