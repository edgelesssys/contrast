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
#
# Besides this full SBOM, three per-category SBOMs are exposed via passthru,
# splitting the shipped software along the confidential-computing boundary so the
# CC TCB can be understood separately from the overall product:
#   - cli                    the command-line tool users run on their workstation
#   - runtimeConfidential    components measured by / running inside the CVM (TCB)
#   - runtimeNonConfidential  host-side, in-cluster components outside the TCB
# There is deliberately no "tooling" SBOM: as a true superset of the three, the
# full matrix SBOM already covers everything else. Each category is defined by a
# small set of root derivations; bombon computes their runtime closure, with the
# vendored SBOMs of those roots merged in exactly as for the full SBOM.
{
  lib,
  buildBom,
  linkFarm,
  stdenv,
  matrix,
  contrastPkgs,
}:

let
  inherit (lib.contrast) collectDerivations buildableOn collectVendoredSbomPackages;

  mkCategorySbom =
    name: roots:
    buildBom
      (linkFarm "contrast-${name}-matrix" (
        buildableOn stdenv.hostPlatform (collectDerivations [ ] roots)
      ))
      {
        extraPaths = collectVendoredSbomPackages [ ] roots;
      };

  # Root derivations per category; a category's SBOM is their runtime closure.
  categories = {
    cli = { inherit (contrastPkgs.contrast) cli; };

    # Everything measured by / running inside the confidential VM: the guest
    # image (rootfs, kernel, initrd), the coordinator and initializer confidential
    # workloads, and the guest firmware.
    #
    # kata.image is a verity blob that discards its internal store references, so
    # its closure is opaque (image, kernel, initrd only). We additionally root at
    # the NixOS system it is built from (passthru.toplevel) to capture the actual
    # guest contents — kata-agent and the full userland — with an intact dependency
    # graph. kata.agent is still listed explicitly so its crate-level vendored SBOM
    # is collected (collectVendoredSbomPackages walks this attrset, not closures).
    runtimeConfidential = {
      inherit (contrastPkgs.contrast) coordinator initializer;
      inherit (contrastPkgs.kata) image agent;
      guestSystem = contrastPkgs.kata.image.passthru.toplevel;
      inherit (contrastPkgs) OVMF-SNP OVMF-TDX;
    };

    runtimeNonConfidential = {
      inherit (contrastPkgs.contrast) nodeinstaller;
      inherit (contrastPkgs) imagepuller imagestore service-mesh;
      inherit (contrastPkgs.kata) runtime runtime-rs;
    };
  };
in

(buildBom matrix {
  extraPaths = collectVendoredSbomPackages [
    "contrastPkgsStatic"
    "contrast-releases"
    "matrix"
    "sbom"
    # cli-release only differs from cli in an embedded genpolicy-settings asset,
    # so its vendored SBOM is identical; skip it to avoid a redundant root.
    "cli-release"
  ] contrastPkgs;
}).overrideAttrs
  (prev: {
    passthru = (prev.passthru or { }) // lib.mapAttrs mkCategorySbom categories;
  })
