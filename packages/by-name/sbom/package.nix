# Copyright 2026 Edgeless Systems GmbH
# SPDX-License-Identifier: BUSL-1.1

{
  lib,
  runCommand,
  closureInfo,
  cyclonedx-cli,
  contrast,
  contrastPkgs,
  matrix,
  sbom-generator,
}:

let
  closure = closureInfo {
    rootPaths = [ matrix ];
  };

  # Deep, language-level SBOMs contributed by individual packages via
  # passthru.sbom (e.g. contrast.contrast, kata.agent, kata.genpolicy,
  # kata.runtime-rs). Any package gains coverage just by setting passthru.sbom.
  collectedSboms = lib.contrast.collectSboms [
    "contrastPkgsStatic"
    "contrast-releases"
    "matrix"
    "sbom"
  ] contrastPkgs;

  # contrast.coordinator and contrast.initializer are outputs of contrast.contrast
  # and inherit its passthru.sbom, so dedupe by store path to merge each SBOM once.
  componentSboms = lib.attrValues (
    lib.listToAttrs (
      map (e: lib.nameValuePair (builtins.unsafeDiscardStringContext "${e.sbom}") e.sbom) collectedSboms
    )
  );
in

runCommand "contrast-sbom.cdx.json"
  {
    nativeBuildInputs = [
      cyclonedx-cli
      sbom-generator
    ];
    passthru = { inherit closure componentSboms; };
  }
  ''
    sbom-generator closure \
      --store-paths ${closure}/store-paths \
      --version ${contrast.contrast.version} \
      --output "$TMPDIR/nix.closure.cdx.json"

    # A flat merge (rather than --hierarchical) is used deliberately: hierarchical
    # merges emit a schema-invalid dependencies graph (duplicate refs) for our
    # inputs, and a flat merge additionally dedupes crates shared between the Rust
    # components. Component provenance is preserved via the dependency graph.
    cyclonedx merge \
      --name contrast \
      --group io.edgeless \
      --version ${contrast.contrast.version} \
      --input-files \
        "$TMPDIR/nix.closure.cdx.json" \
        ${lib.concatMapStringsSep " " (s: ''"${s}"'') componentSboms} \
      --output-file "$out" \
      --output-format json
  ''
